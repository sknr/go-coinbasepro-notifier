package app

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/NicoNex/echotron/v3"
	"github.com/foxever/sqlite"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/sknr/go-coinbasepro-notifier/internal/database"
	"github.com/sknr/go-coinbasepro-notifier/internal/logger"
	"github.com/sknr/go-coinbasepro-notifier/internal/telegram"
	"github.com/sknr/go-coinbasepro-notifier/internal/updater"
	"github.com/sknr/go-coinbasepro-notifier/internal/utils"
	"github.com/sknr/go-coinbasepro-notifier/internal/watcher"
	"gorm.io/gorm"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	sessionName      = "coinbasepro-notifier"
	maxNumberOfUsers = 25 // Maximum number of users supported
	version          = "v1.0.3"
)

var app *App

type App struct {
	db            *gorm.DB
	sessionStore  *sessions.CookieStore
	telegramToken string
	watchers      map[string]*watcher.CoinbaseProWatcher
	updater       *updater.Updater
	mu            sync.Mutex
}

type TelegramUser struct {
	ID              string
	Alias           string
	FirstName       string
	LastName        string
	PhotoURL        string
	IsAuthenticated bool
}

func New() *App {
	a := new(App)
	a.updater = updater.New()

	authKeyOne := securecookie.GenerateRandomKey(64)
	encryptionKeyOne := securecookie.GenerateRandomKey(32)

	a.sessionStore = sessions.NewCookieStore(
		authKeyOne,
		encryptionKeyOne,
	)

	a.sessionStore.Options = &sessions.Options{
		MaxAge:   3600,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
	}
	// Register User for session storage
	gob.Register(TelegramUser{})

	// Set the telegram token
	utils.CheckEnvVars("TELEGRAM_TOKEN", "DATABASE_FILE")
	a.telegramToken = os.Getenv("TELEGRAM_TOKEN")

	// Create clients map
	a.watchers = make(map[string]*watcher.CoinbaseProWatcher)

	// Initialize database
	var err error
	a.db, err = gorm.Open(sqlite.Open(os.Getenv("DATABASE_FILE")), &gorm.Config{})
	logger.LogErrorIfExists(err)
	// Create table if not exists
	logger.LogErrorIfExists(a.db.AutoMigrate(&database.UserSettings{}))

	app = a
	return app
}

// Start main function to start the coinbase notifier server and
// the websockets connection for the registered clients
func (a *App) Start() {
	// Start websocket connections for each client
	a.startWatchers()
	// Create router and setup routes
	logger.LogInfo("Starting server at port 8080")
	a.startServer()
}

// startServer creates the necessary routes for the http.server and registers the bot listing for updates on the webhook handler
func (a *App) startServer() {
	router := mux.NewRouter()
	router.HandleFunc("/", a.homeHandler)
	router.HandleFunc("/form/settings", a.settingsHandler)
	router.HandleFunc("/form/delete-profile", a.deleteHandler)
	router.HandleFunc("/login", a.loginHandler)
	router.HandleFunc("/logout", a.logoutHandler)
	// Add static file server
	fileServer := http.FileServer(http.Dir("./static"))
	router.PathPrefix("/").Handler(http.StripPrefix("/", fileServer))

	termChan := make(chan os.Signal, 1) // Channel for terminating the app via os.Interrupt signal
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	// Capture the interrupt signal for app termination handling
	dsp := echotron.NewDispatcher(os.Getenv("TELEGRAM_TOKEN"), newBot)
	server := &http.Server{Addr: ":8080", Handler: router}
	// Set custom http.Server
	dsp.SetHTTPServer(server)

	go func() {
		<-termChan
		logger.LogInfo("SIGTERM received -> Shutdown process initiated")
		a.updater.Stop()
		logger.LogErrorIfExists(server.Shutdown(context.Background()))
	}()

	logger.LogInfof("Starting telegram bot server at %q", server.Addr)
	// Start Webserver with provided webhook
	logger.LogErrorIfExists(dsp.ListenWebhook("https://notifier.bot.apperia.de/webhook"))
}

// startWatchers creates a websocket connection for each user
func (a *App) startWatchers() {
	a.mu.Lock()
	defer a.mu.Unlock()
	var userSettings []database.UserSettings
	a.db.Where("active = ?", true).Find(&userSettings)
	for _, settings := range userSettings {
		// Skip subscription if settings are missing
		if settings.APIKey == "" {
			continue
		}
		// Create the client
		a.watchers[settings.TelegramID] = watcher.New(settings, a.updater)
		// Start watching for user related order updates
		go a.watchers[settings.TelegramID].Start()
		// We need to sleep in order to not hit the coinbase pro api limits
		time.Sleep(1 * time.Second)
	}
}

/************/
/* Database */
/************/

// createOrUpdateUser creates a new user or updates a given user if already exists
func (a *App) createOrUpdateUser(user TelegramUser) {
	var settings = database.UserSettings{}
	a.db.First(&settings, user.ID)
	if settings.TelegramID == "" {
		// New user will be created
		telegram.SendAdminPushMessage(fmt.Sprintf("New user has successfully registered:\n%#v", user))
		logger.LogInfof("Created new user: %#v", user)
	}
	settings.TelegramID = user.ID
	settings.FirstName = user.FirstName
	settings.LastName = user.LastName
	settings.Username = user.Alias
	settings.PhotoURL = user.PhotoURL
	a.db.Save(&settings)
}

// getTotalNumberOfActiveUsers get all active users
func (a *App) getTotalNumberOfActiveUsers() int {
	var number int
	a.db.Raw("SELECT COUNT(telegram_id) FROM user_settings").Scan(&number)

	return number
}

// getUserSettings get all user settings with specified active status.
func (a *App) getUserSettings(active bool) []database.UserSettings {
	var userSettings []database.UserSettings
	a.db.Where("active = ?", active).Find(&userSettings)

	return userSettings
}

// getAllUserSettings get all user settings.
func (a *App) getAllUserSettings() []database.UserSettings {
	var userSettings []database.UserSettings
	a.db.Find(&userSettings)

	return userSettings
}

/************/
/* Handlers */
/************/

// loginHandler handles the login via telegram login widget
func (a *App) loginHandler(w http.ResponseWriter, r *http.Request) {
	submittedHash := r.URL.Query().Get("hash")
	params := getQueryParams(r, []string{"auth_date", "first_name", "last_name", "photo_url", "id", "username"})
	var sortedParams []string
	for key, val := range params {
		if val != "" {
			sortedParams = append(sortedParams, key+"="+val)
		}
	}
	sort.Strings(sortedParams)
	checkString := strings.Join(sortedParams, "\n")

	// Hash the secret
	hs := sha256.New()
	hs.Write([]byte(a.telegramToken))
	// Hash the checkString with the hashed secret
	h := hmac.New(sha256.New, hs.Sum(nil))
	h.Write([]byte(checkString))
	sha := hex.EncodeToString(h.Sum(nil))

	logger.LogDebugf("Raw-Query: %s", r.URL.RawQuery)
	logger.LogDebugf("QueryParams: %#v", params)
	logger.LogDebugf("SortedParams: %#v", sortedParams)
	logger.LogDebugf("Check-String: %s", checkString)
	logger.LogDebugf("Checksum SHA <> submitted SHA => %s <> %s", sha, submittedHash)

	if sha != submittedHash {
		logger.LogInfo("Login failed!", params["id"])
		renderTemplate(w, "error", struct{ ErrorMessage string }{"Checksum-Error! Someone seems to try nasty stuff..."})
		return
	}

	// Login successful
	session, _ := a.sessionStore.Get(r, sessionName)
	user := getUser(session)
	user.ID = params["id"]
	user.FirstName = params["first_name"]
	user.LastName = params["last_name"]
	user.Alias = params["username"]
	user.PhotoURL = params["photo_url"]
	user.IsAuthenticated = true
	session.Values["user"] = user
	logger.LogErrorIfExists(session.Save(r, w))

	a.createOrUpdateUser(user)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// logoutHandler handles the user logout
func (a *App) logoutHandler(w http.ResponseWriter, r *http.Request) {
	// Remove the session
	session, _ := a.sessionStore.Get(r, sessionName)
	session.Options.MaxAge = -1
	_ = session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// homeHandler handles the user profile page
func (a *App) homeHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := a.sessionStore.Get(r, sessionName)
	user := getUser(session)
	var userSettings = database.UserSettings{}
	a.db.First(&userSettings, user.ID)

	// We currently support only maxNumberOfUsers in parallel
	if a.getTotalNumberOfActiveUsers() >= maxNumberOfUsers {
		renderTemplate(w, "error", struct{ ErrorMessage string }{"Maximum number of users reached! Please try again later"})
		return
	}

	if !user.IsAuthenticated {
		renderTemplate(w, "index", nil)
		return
	}
	renderTemplate(w, "profile", userSettings)
}

// settingsHandler receives the html form post values and updates the user settings
func (a *App) settingsHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		logger.LogError(err)
		renderTemplate(w, "error", struct{ ErrorMessage string }{"Could not parse form"})
		return
	}

	if r.Method != http.MethodPost {
		renderTemplate(w, "error", struct{ ErrorMessage string }{"Method not allowed"})
		return
	}

	session, _ := a.sessionStore.Get(r, sessionName)
	user := getUser(session)
	if !user.IsAuthenticated {
		renderTemplate(w, "error", struct{ ErrorMessage string }{"Access denied"})
		return
	}

	var userSettings = database.UserSettings{}
	a.db.First(&userSettings, user.ID)
	userSettings.APIKey = r.FormValue("key")
	userSettings.APIPassphrase = r.FormValue("passphrase")
	userSettings.APISecret = r.FormValue("secret")
	a.db.Save(&userSettings)

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.watchers[user.ID] != nil {
		// Close the existing client
		a.watchers[user.ID].Stop()
	}
	// Only start a new watcher if user is active.
	if userSettings.Active {
		a.watchers[user.ID] = watcher.New(userSettings, a.updater)
		// Start watching for user related order updates
		go a.watchers[user.ID].Start()
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// deleteHandler removes the user from database and performs logout
func (a *App) deleteHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := a.sessionStore.Get(r, sessionName)
	user := getUser(session)
	if !user.IsAuthenticated {
		renderTemplate(w, "error", struct{ ErrorMessage string }{"Access denied"})
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.watchers[user.ID] != nil {
		// Close the existing client
		a.watchers[user.ID].Stop()
		delete(a.watchers, user.ID)
	}
	a.db.Delete(&database.UserSettings{}, user.ID)
	telegram.SendAdminPushMessage(fmt.Sprintf("User with ID (%s) has deleted his/her profile:\n%#v", user.ID, user))
	logger.LogInfof("User with ID (%s) has deleted his/her profile:\n%#v", user.ID, user)

	// Call logout handler to remove session and redirect user to login page
	a.logoutHandler(w, r)
}

// enableUser sets the active flag to true and starts the watcher
func (a *App) enableUser(telegramID string) {
	var userSettings database.UserSettings
	err := a.db.Where("telegram_id = ?", telegramID).First(&userSettings).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}
	userSettings.Active = true
	a.db.Save(&userSettings)

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.watchers[telegramID] != nil {
		// Close the existing client
		a.watchers[telegramID].Stop()
	}
	a.watchers[telegramID] = watcher.New(userSettings, a.updater)
	// Start watching for user related order updates
	go a.watchers[telegramID].Start()
}

// disableUser sets the active flag to false and stops the watcher
func (a *App) disableUser(telegramID string) {
	var userSettings database.UserSettings
	err := a.db.Where("telegram_id = ?", telegramID).First(&userSettings).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}
	userSettings.Active = false
	a.db.Save(&userSettings)

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.watchers[telegramID] != nil {
		// Close the existing client
		a.watchers[telegramID].Stop()
		delete(a.watchers, telegramID)
	}
}

// deleteUser deletes an user from database
func (a *App) deleteUser(telegramID string) {
	var userSettings database.UserSettings
	err := a.db.Where("telegram_id = ?", telegramID).First(&userSettings).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.watchers[telegramID] != nil {
		// Close the existing client
		a.watchers[telegramID].Stop()
		delete(a.watchers, telegramID)
	}
	a.db.Delete(&userSettings)
	logger.LogInfof("User with ID (%s) has been deleted:\n%#v", telegramID, userSettings)
}

// getQueryParams retrieves the given parameter list from the query
func getQueryParams(r *http.Request, keys []string) map[string]string {
	var sortedParams = make(map[string]string)
	for _, key := range keys {
		val := r.URL.Query().Get(key)
		sortedParams[key] = val
	}

	return sortedParams
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	t, _ := template.ParseFiles("static/" + tmpl + ".html")
	logger.LogErrorIfExists(t.Execute(w, data))
}

// getUser returns a user from session s. on error returns an empty user
func getUser(s *sessions.Session) TelegramUser {
	val := s.Values["user"]
	user, ok := val.(TelegramUser)
	if !ok {
		return TelegramUser{IsAuthenticated: false}
	}
	return user
}
