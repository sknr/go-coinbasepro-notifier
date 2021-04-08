package telegram

import (
	"fmt"
	"github.com/sknr/go-coinbasepro-notifier/internal/logger"
	"github.com/yanzay/tbot/v2"
	"os"
)

// Send a telegram message to the user with given chatID
func SendPushMessage(chatID, message string) {
	botToken := os.Getenv("TELEGRAM_TOKEN")
	tc := tbot.New(botToken).Client()
	if message != "" {
		_, err := tc.SendMessage(chatID, message)
		logger.LogErrorIfExists(err)
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
	tc := tbot.New(botToken).Client()
	if message != "" {
		_, err := tc.SendMessage(adminChatID, message)
		logger.LogErrorIfExists(err)
	}
}

// SendAdminPushMessageWhenPanic sends a push message on application panic
func SendAdminPushMessageWhenPanic() {
	if r := recover(); r != nil {
		SendAdminPushMessage(fmt.Sprintf("App panicked!\n%s", r))
	}
}
