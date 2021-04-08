package telegram

import (
	"fmt"
	"github.com/sknr/go-coinbasepro-notifier/internal/logger"
	"github.com/yanzay/tbot/v2"
	"os"
)

// CreateBot creates a new telegram bot
func CreateBot() *tbot.Server {
	bot := tbot.New(os.Getenv("TELEGRAM_TOKEN"), tbot.WithWebhookForCustomServer("https://notifier.bot.apperia.de/webhook"))
	c := bot.Client()

	loginButton := makeLoginButton()
	var err error
	bot.HandleMessage("setup", func(m *tbot.Message) {
		_, err = c.SendMessage(m.Chat.ID, fmt.Sprintf("Hi %s,\n🤝 welcome to Coinbase Pro Notifier. Please click the setup button below to complete the setup in order to get informed about your Coinbase Pro order updates",m.Chat.FirstName), tbot.OptInlineKeyboardMarkup(loginButton))
		logger.LogErrorIfExists(err)
	})

	bot.HandleMessage("/start", func(m *tbot.Message) {
		_, err = c.SendMessage(m.Chat.ID, fmt.Sprintf("Hi %s,\n🤝 welcome to Coinbase Pro Notifier. Please click the setup button below to complete the setup in order to get informed about your Coinbase Pro order updates",m.Chat.FirstName), tbot.OptInlineKeyboardMarkup(loginButton))
		logger.LogErrorIfExists(err)
	})

	bot.HandleMessage("ping", func(m *tbot.Message) {
		_, err = c.SendMessage(m.Chat.ID, "pong")
		logger.LogErrorIfExists(err)
	})

	return bot
}

/*
 * makeLoginButton creates an telegram login button for easy registering
 * on the webserver via telegram login
 */
func makeLoginButton() *tbot.InlineKeyboardMarkup {
	loginButton := tbot.InlineKeyboardButton{
		Text: "Open setup page",
		LoginURL: &tbot.LoginURL{
			URL: "https://notifier.bot.apperia.de/login",
		},
	}

	return &tbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]tbot.InlineKeyboardButton{
			[]tbot.InlineKeyboardButton{loginButton},
		},
	}
}
