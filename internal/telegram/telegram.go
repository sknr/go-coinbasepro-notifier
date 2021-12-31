package telegram

import (
	"fmt"
	"github.com/NicoNex/echotron/v3"
	"github.com/sknr/go-coinbasepro-notifier/internal/logger"
	"os"
	"runtime/debug"
	"strconv"
)

// SendPushMessage sends a telegram message to the user with given chatID
func SendPushMessage(chatID, message string) {
	botToken := os.Getenv("TELEGRAM_TOKEN")
	api := echotron.NewAPI(botToken)
	cID, err := strconv.ParseInt(chatID, 10, 64)
	logger.LogErrorIfExists(err)
	if message != "" {
		_, err = api.SendMessage(message, cID, nil)
		logger.LogErrorIfExists(err, chatID)
	}
}

// SendAdminPushMessage sends an telegram message to the admin only
func SendAdminPushMessage(message string) {
	adminChatID := os.Getenv("TELEGRAM_ADMIN_CHAT_ID")
	if adminChatID == "" {
		logger.LogWarn("Missing env var \"TELEGRAM_ADMIN_CHAT_ID\" -> Cannot send admin push message")
		return
	}
	botToken := os.Getenv("TELEGRAM_TOKEN")
	api := echotron.NewAPI(botToken)
	cID, err := strconv.ParseInt(adminChatID, 10, 64)
	logger.LogErrorIfExists(err)
	if message != "" {
		_, err = api.SendMessage(message, cID, nil)
		logger.LogErrorIfExists(err, adminChatID)
	}
}

// SendAdminPushMessageWhenPanic sends a push message on application panic
func SendAdminPushMessageWhenPanic() {
	if err := recover(); err != nil {
		logger.LogWarnf("App panicked!\n%s", err)
		logger.LogWarn("Stack Trace:")
		debug.PrintStack()
		SendAdminPushMessage(fmt.Sprintf("App panicked!\n%s", err))
	}
}
