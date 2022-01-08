package app

import (
	"fmt"
	"github.com/NicoNex/echotron/v3"
	"github.com/sknr/go-coinbasepro-notifier/internal/database"
	"github.com/sknr/go-coinbasepro-notifier/internal/logger"
	"github.com/sknr/go-coinbasepro-notifier/internal/telegram"
	"os"
	"strconv"
	"strings"
)

type bot struct {
	chatID      int64
	lastCommand string
	echotron.API
}

const (
	cmdStart       = "/start"
	cmdShowVersion = "/version"
	cmdEnableUser  = "/enable_user"
	cmdDisableUser = "/disable_user"
	cmdDeleteUser  = "/delete_user"
)

func newBot(chatID int64) echotron.Bot {
	return &bot{
		chatID:      chatID,
		lastCommand: "",
		API:         echotron.NewAPI(os.Getenv("TELEGRAM_TOKEN")),
	}
}

func (b *bot) Update(update *echotron.Update) {
	if update.Message != nil {
		b.handleMessage(update.Message)
	}
	if update.CallbackQuery != nil {
		b.handleMessage(update.CallbackQuery)
	}
}

func (b *bot) handleCommand(msg *echotron.Message, data string) {
	if isCommand(msg) {
		logger.LogInfof("[%s:%d] New Command: %s", msg.Chat.FirstName, msg.Chat.ID, msg.Text)
		b.lastCommand = msg.Text
		b.handleMessageData(msg, data)
		return
	}
	if b.lastCommand != "" && msg != nil {
		logger.LogInfof("[%s:%d] LastCommand: %s | Data: %s", msg.Chat.FirstName, msg.Chat.ID, b.lastCommand, data)
		b.handleMessageData(msg, data)
		return
	}
	if msg != nil {
		logger.LogInfof("[%s:%d] Message: %s", msg.Chat.FirstName, msg.Chat.ID, msg.Text)
	}
}

func (b *bot) handleMessage(msg interface{}) {
	switch t := msg.(type) {
	case *echotron.Message:
		logger.LogInfof("Message received")
		b.handleCommand(t, "")
	case *echotron.CallbackQuery:
		logger.LogInfof("Callback message received! Data: %s", t.Data)
		b.handleCommand(t.Message, t.Data)
	default:
		logger.LogInfof("Unknown message type: %t", t)
	}
}

func (b *bot) handleMessageData(msg *echotron.Message, data string) {
	var err error
	switch b.lastCommand {
	case cmdStart:
		b.sendWelcomeMessage(msg)
	case cmdEnableUser:
		if !isAdmin(b.chatID) {
			logger.LogWarnf("[%s:%d] Non admin user tries to run command: %s", msg.Chat.FirstName, msg.Chat.ID, b.lastCommand)
			telegram.SendAdminPushMessage(fmt.Sprintf("[%s:%d] Non admin users tries to run command: %s", msg.Chat.FirstName, msg.Chat.ID, b.lastCommand))
			break
		}
		if data == "" {
			us := app.getUserSettings(false)
			_, err = b.SendMessage("Enable user: "+data, b.chatID, &echotron.MessageOptions{
				ReplyMarkup: createInlineButtons(us),
			})
			logger.LogErrorIfExists(err, b.chatID)
			return
		} else {
			_, err = b.DeleteMessage(b.chatID, msg.ID)
			logger.LogErrorIfExists(err, b.chatID)
			app.enableUser(data)
		}
	case cmdDisableUser:
		if !isAdmin(b.chatID) {
			logger.LogWarnf("[%s:%d] Non admin users tries to run command: %s", msg.Chat.FirstName, msg.Chat.ID, b.lastCommand)
			telegram.SendAdminPushMessage(fmt.Sprintf("[%s:%d] Non admin users tries to run command: %s", msg.Chat.FirstName, msg.Chat.ID, b.lastCommand))
			break
		}
		if data == "" {
			us := app.getUserSettings(true)
			_, err = b.SendMessage("Disable user: "+data, b.chatID, &echotron.MessageOptions{
				ReplyMarkup: createInlineButtons(us),
			})
			logger.LogErrorIfExists(err, b.chatID)
			return
		} else {
			_, err = b.DeleteMessage(b.chatID, msg.ID)
			logger.LogErrorIfExists(err, b.chatID)
			app.disableUser(data)
		}
	case cmdDeleteUser:
		if !isAdmin(b.chatID) {
			logger.LogWarnf("[%s:%d] Non admin users tries to run command: %s", msg.Chat.FirstName, msg.Chat.ID, b.lastCommand)
			telegram.SendAdminPushMessage(fmt.Sprintf("[%s:%d] Non admin users tries to run command: %s", msg.Chat.FirstName, msg.Chat.ID, b.lastCommand))
			break
		}
		if data == "" {
			us := app.getAllUserSettings()
			_, err = b.SendMessage("Delete user: "+data, b.chatID, &echotron.MessageOptions{
				ReplyMarkup: createInlineButtons(us),
			})
			logger.LogErrorIfExists(err, b.chatID)
			return
		} else {
			_, err = b.DeleteMessage(b.chatID, msg.ID)
			logger.LogErrorIfExists(err, b.chatID)
			app.deleteUser(data)
		}
	case cmdShowVersion:
		_, err = b.SendMessage(version, b.chatID, nil)
		logger.LogErrorIfExists(err, b.chatID)
	default:
		logger.LogInfof("[%s:%d] Unknown command: %s", msg.Chat.FirstName, msg.Chat.ID, b.lastCommand)
	}
	b.lastCommand = ""
}

func isAdmin(id int64) bool {
	return strconv.FormatInt(id, 10) == os.Getenv("TELEGRAM_ADMIN_CHAT_ID")
}

func isCommand(message *echotron.Message) bool {
	return message != nil && strings.HasPrefix(message.Text, "/")
}

func (b *bot) sendWelcomeMessage(msg *echotron.Message) {
	_, err := b.SendMessage(fmt.Sprintf("Hi %s,\n🤝 welcome to Coinbase Pro Notifier. Please click the setup button below to complete the setup in order to get informed about your Coinbase Pro order updates", msg.Chat.FirstName), msg.Chat.ID, &echotron.MessageOptions{
		ReplyMarkup: echotron.InlineKeyboardMarkup{
			InlineKeyboard: [][]echotron.InlineKeyboardButton{
				{
					{
						Text: "Open setup page",
						URL:  "",
						LoginURL: &echotron.LoginURL{
							URL: "https://notifier.bot.apperia.de/login",
						},
					},
				},
			},
		},
	})
	logger.LogErrorIfExists(err, b.chatID)
}

func createInlineButtons(settings []database.UserSettings) echotron.InlineKeyboardMarkup {
	var (
		row     []echotron.InlineKeyboardButton
		buttons [][]echotron.InlineKeyboardButton
	)
	for i, user := range settings {
		btn := echotron.InlineKeyboardButton{
			Text:         fmt.Sprintf("%s (%s)", user.FirstName, user.TelegramID),
			CallbackData: user.TelegramID,
		}
		row = append(row, btn)
		if (i+1)%2 == 0 {
			buttons = append(buttons, row)
			row = []echotron.InlineKeyboardButton{}
		}
	}
	buttons = append(buttons, row)

	return echotron.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}
}
