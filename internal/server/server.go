package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/preichenberger/go-coinbasepro/v2"
	. "github.com/sknr/go-coinbasepro-notifier/internal"
	"github.com/sknr/go-coinbasepro-notifier/internal/client"
	"github.com/sknr/go-coinbasepro-notifier/internal/database"
	"github.com/sknr/go-coinbasepro-notifier/internal/logger"
	"github.com/sknr/go-coinbasepro-notifier/internal/telegram"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"
)

const (
	sessionName      = "coinbasepro-notifier"
	maxNumberOfUsers = 100 // Maximum number of users supported
)

var (
	db            *gorm.DB
	productIDs    []string
	sessionStore  *sessions.CookieStore
	telegramToken string
	clients       map[string]*client.CoinbaseProClient
)

type TelegramUser struct {
	ID              string
	Alias           string
	FirstName       string
	LastName        string
	PhotoURL        string
	IsAuthenticated bool
}

func initialize() {
	authKeyOne := securecookie.GenerateRandomKey(64)
	encryptionKeyOne := securecookie.GenerateRandomKey(32)

	sessionStore = sessions.NewCookieStore(
		authKeyOne,
		encryptionKeyOne,
	)

	sessionStore.Options = &sessions.Options{
		MaxAge:   3600,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
	}
	// Register User for session storage
	gob.Register(TelegramUser{})

	// Set the telegram token
	CheckEnvVars("TELEGRAM_TOKEN", "DATABASE_FILE")
	telegramToken = os.Getenv("TELEGRAM_TOKEN")

	// Create clients map
	clients = make(map[string]*client.CoinbaseProClient)

	// Initialize database
	var err error
	db, err = gorm.Open(sqlite.Open(os.Getenv("DATABASE_FILE")), &gorm.Config{})
	logger.LogErrorIfExists(err)
	// Create table if not exists
	logger.LogErrorIfExists(db.AutoMigrate(&database.UserSettings{}))

	// Start websocket connections for each client
	go initializeExistingClients()
}

/*
 * Start main function to start the coinbase notifier
 * server and the websockets connection for the registered clients
 */
func Start() {
	// Send a push message to the admin in case the app panicked
	defer telegram.SendAdminPushMessageWhenPanic()
	// Initialize server settings
	initialize()

	// Create telegram bot and get webhook handler
	bot := telegram.CreateBot()
	// Start Telegram bot
	go func() {
		logger.LogErrorIfExists(bot.Start())
	}()

	// Create router and setup routes
	router := mux.NewRouter()
	router.HandleFunc("/", homeHandler)
	router.HandleFunc("/form/settings", settingsHandler)
	router.HandleFunc("/form/delete-profile", deleteHandler)
	router.HandleFunc("/webhook", bot.GetWebhookHandler())
	router.HandleFunc("/login", loginHandler)
	router.HandleFunc("/logout", logoutHandler)
	// Add static file server
	fileServer := http.FileServer(http.Dir("./static"))
	router.PathPrefix("/").Handler(http.StripPrefix("/", fileServer))

	var termChan chan os.Signal // Channel for terminating the app via os.Interrupt signal
	termChan = make(chan os.Signal, 1)
	// Capture the interrupt signal for app termination handling
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	httpServer := &http.Server{Addr: ":8080", Handler: router}
	go func() {
		<-termChan
		bot.Stop()
		logger.LogInfo("SIGTERM received -> Shutdown process initiated")
		logger.LogErrorIfExists(httpServer.Shutdown(context.Background()))
	}()

	logger.LogInfo("Starting server at port 8080")
	err := httpServer.ListenAndServe()
	if err != nil {
		logger.LogInfo(err.Error())
	}
}

// initializeExistingClients creates a websocket connection for each user
func initializeExistingClients() {
	cbp := client.NewCoinbaseProClient(database.UserSettings{}, &client.CoinbaseProClientConfig{})
	defer cbp.Close()

	var err error
	productIDs, err = cbp.GetAllAvailableProductIDs()
	if HasError(err) {
		logger.LogError(err)
		panic("Cannot retrieve current list of product ids from Coinbase Pro. %s",)
	}
	logger.LogInfof("ProductIDs: %s", strings.Join(productIDs, ","))

	var userSettings []database.UserSettings
	db.Find(&userSettings)
	for _, settings := range userSettings {
		config := client.CoinbaseProClientConfig{
			ClientConfig: coinbasepro.ClientConfig{
				BaseURL:    os.Getenv("COINBASE_PRO_BASEURL"),
				Key:        settings.APIKey,
				Passphrase: settings.APIPassphrase,
				Secret:     settings.APISecret,
			},
		}

		// Skip subscription if settings are missing
		if settings.APIKey == "" {
			continue
		}

		// Create the client
		clients[settings.TelegramID] = client.NewCoinbaseProClient(settings, &config)

		channels := []coinbasepro.MessageChannel{
			{
				Name:       client.ChannelTypeUser,
				ProductIds: productIDs, //productIDsToSlice(settings.ProductIDs),
			},
		}
		// Start watching for user related order updates
		go clients[settings.TelegramID].Watch(channels)
		// We need to sleep in order to not hit the coinbase pro api limits
		time.Sleep(1 * time.Second)
	}
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

// createOrUpdateUser creates a new user or updates a given user if already exists
func createOrUpdateUser(user TelegramUser) {
	var settings = database.UserSettings{}
	db.First(&settings, user.ID)
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
	db.Save(&settings)
}



func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	t, _ := template.ParseFiles("static/" + tmpl + ".html")
	logger.LogErrorIfExists(t.Execute(w, data))
}

// getUser returns a user from session s. on error returns an empty user
func getUser(s *sessions.Session) TelegramUser {
	val := s.Values["user"]
	var user = TelegramUser{}
	user, ok := val.(TelegramUser)
	if !ok {
		return TelegramUser{IsAuthenticated: false}
	}
	return user
}

// getTotalNumberOfActiveUsers get all active users
func getTotalNumberOfActiveUsers() int {
	var number int
	db.Raw("SELECT COUNT(telegram_id) FROM user_settings").Scan(&number)

	return number
}

/************/
/* Handlers */
/************/


// loginHandler handles the login via telegram login widget
func loginHandler(w http.ResponseWriter, r *http.Request) {
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
	hs.Write([]byte(telegramToken))
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
	session, _ := sessionStore.Get(r, sessionName)
	user := getUser(session)
	user.ID = params["id"]
	user.FirstName = params["first_name"]
	user.LastName = params["last_name"]
	user.Alias = params["username"]
	user.PhotoURL = params["photo_url"]
	user.IsAuthenticated = true
	session.Values["user"] = user
	logger.LogErrorIfExists(session.Save(r, w))

	createOrUpdateUser(user)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}


// logoutHandler handles the user logout
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	// Remove the session
	session, _ := sessionStore.Get(r, sessionName)
	session.Options.MaxAge = -1
	_ = session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// homeHandler handles the user profile page
func homeHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, sessionName)
	user := getUser(session)
	var userSettings = database.UserSettings{}
	db.First(&userSettings, user.ID)

	// We currently support only maxNumberOfUsers in parallel
	if getTotalNumberOfActiveUsers() >= maxNumberOfUsers {
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
func settingsHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}

	if r.Method != http.MethodPost {
		renderTemplate(w, "error", struct{ ErrorMessage string }{"Method not allowed"})
		return
	}

	session, _ := sessionStore.Get(r, sessionName)
	user := getUser(session)
	if !user.IsAuthenticated {
		renderTemplate(w, "error", struct{ ErrorMessage string }{"Access denied"})
		return
	}

	var userSettings = database.UserSettings{}
	db.First(&userSettings, user.ID)
	userSettings.APIKey = r.FormValue("key")
	userSettings.APIPassphrase = r.FormValue("passphrase")
	userSettings.APISecret = r.FormValue("secret")
	//userSettings.ProductIDs = r.FormValue("product_ids")
	db.Save(&userSettings)

	if clients[user.ID] != nil {
		// Close the existing client
		clients[user.ID].Close()
	}

	// Create new client
	config := client.CoinbaseProClientConfig{
		ClientConfig: coinbasepro.ClientConfig{
			BaseURL:    os.Getenv("COINBASE_PRO_BASEURL"),
			Key:        userSettings.APIKey,
			Passphrase: userSettings.APIPassphrase,
			Secret:     userSettings.APISecret,
		},
	}

	cbp := client.NewCoinbaseProClient(userSettings, &config)
	clients[user.ID] = cbp

	channels := []coinbasepro.MessageChannel{
		{
			Name:       client.ChannelTypeUser,
			ProductIds: productIDs,
		},
	}
	// Start watching for user related order updates
	go clients[user.ID].Watch(channels)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// deleteHandler removes the user from database and performs logout
func deleteHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, sessionName)
	user := getUser(session)
	if !user.IsAuthenticated {
		renderTemplate(w, "error", struct{ ErrorMessage string }{"Access denied"})
		return
	}

	if clients[user.ID] != nil {
		// Close the existing client
		clients[user.ID].Close()
		delete(clients, user.ID)
	}
	db.Delete(&database.UserSettings{}, user.ID)
	telegram.SendAdminPushMessage(fmt.Sprintf("User with ID (%s) has deleted his/her profile:\n%#v", user.ID, user))
	logger.LogInfof("User with ID (%s) has deleted his/her profile:\n%#v", user.ID, user)

	// Call logout handler to remove session and redirect user to login page
	logoutHandler(w, r)
}