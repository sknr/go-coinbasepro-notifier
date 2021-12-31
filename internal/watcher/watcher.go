package watcher

import (
	"fmt"
	"github.com/preichenberger/go-coinbasepro/v2"
	"github.com/recws-org/recws"
	"github.com/sknr/go-coinbasepro-notifier/internal/database"
	"github.com/sknr/go-coinbasepro-notifier/internal/logger"
	"github.com/sknr/go-coinbasepro-notifier/internal/telegram"
	"github.com/sknr/go-coinbasepro-notifier/internal/updater"
	"github.com/sknr/go-coinbasepro-notifier/internal/utils"
	"os"
	"time"
)

const (
	CoinbaseProURL          = "https://api.pro.coinbase.com"
	CoinbaseProWebSocketURL = "wss://ws-feed.pro.coinbase.com"
)

type CoinbaseProWatcher struct {
	client       *coinbasepro.Client
	ws           *recws.RecConn
	userSettings database.UserSettings // Current user settings
	channel      channel
	updater      *updater.Updater
}

type channel struct {
	order     chan OrderMessage
	terminate chan struct{}
}

func New(userSettings database.UserSettings, updater *updater.Updater) *CoinbaseProWatcher {
	c := coinbasepro.NewClient()

	return &CoinbaseProWatcher{
		client:       c,
		ws:           nil,
		updater:      updater,
		userSettings: userSettings,
	}
}

func (w *CoinbaseProWatcher) Start() {
	wsURL := os.Getenv("COINBASE_PRO_WEBSOCKET_URL")
	if wsURL == "" {
		wsURL = CoinbaseProWebSocketURL
	}

	// Initialize channels
	w.channel.order = make(chan OrderMessage, 5)
	w.channel.terminate = make(chan struct{})

	w.ws = recws.New(
		recws.WithKeepAliveTimeout(10*time.Second),
		recws.WithReconnectInterval(2*time.Second, 256*time.Second, 2),
		recws.WithSubscribeHandler(w.subscribeHandler),
		//recws.WithVerbose(),
	)

	// Create new WebSocket connection
	w.ws.Dial(wsURL, nil)

	// Block until app termination (CTRL-C) or we receive a message on the close channel
	for {
		select {
		case <-w.channel.terminate:
			w.ws.Shutdown()
			logger.LogInfof("Closing client with ID %q", w.userSettings.TelegramID)
			return
		case orderMessage := <-w.channel.order:
			telegram.SendPushMessage(w.userSettings.TelegramID, orderMessage.String())
		}
	}
}

func (w *CoinbaseProWatcher) Stop() {
	close(w.channel.terminate)
}

func (w *CoinbaseProWatcher) handleWebSocketMessage(message coinbasepro.Message) {
	switch message.Type {
	case MessageTypeActivate, MessageTypeChange, MessageTypeDone, MessageTypeMatch, MessageTypeOpen, MessageTypeReceived:
		logger.LogInfo("Order-Message", w.userSettings.TelegramID, message)
		w.handleOrderMessage(message)
	case MessageTypeError:
		logger.LogWarn("ErrorMessage", w.userSettings.TelegramID, message.Message)
		if message.Message == "Authentication Failed" {
			telegram.SendPushMessage(w.userSettings.TelegramID, "Coinbase Pro authentication failed. Please check your API-Settings, in order to get informed about your order changes.")
		}
		telegram.SendAdminPushMessage(fmt.Sprintf("Received an error message for user %s (%s)\nErrorMessage: %s", w.userSettings.FirstName, w.userSettings.TelegramID, message.Message))
	case MessageTypeSubscriptions:
		logger.LogInfo("Successfully subscribed to channels", w.userSettings.TelegramID, message.Channels)
	case MessageTypeStatus:
		logger.LogInfo("Status-Message", w.userSettings.TelegramID, message)
	default:
		logger.LogInfof("Received message of unknown type %q", message.Type)
		logger.LogInfof("Message: %#v", message)
	}
}

// handleOrderMessage converts a coinbasepro.Message into an OrderMessage
func (w *CoinbaseProWatcher) handleOrderMessage(message coinbasepro.Message) {
	messageTime := message.Time.Time()
	orderMessage := OrderMessage{
		Type:          message.Type,
		Time:          &messageTime,
		ProductID:     message.ProductID,
		Sequence:      message.Sequence,
		OrderID:       message.OrderID,
		Funds:         message.Funds,
		NewFunds:      message.NewFunds,
		OldFunds:      message.OldFunds,
		Side:          message.Side,
		OrderType:     message.OrderType,
		Price:         message.Price,
		RemainingSize: message.RemainingSize,
		Size:          message.Size,
		NewSize:       message.NewSize,
		OldSize:       message.OldSize,
		Reason:        message.Reason,
		TradeID:       message.TradeID,
		MakerOrderID:  message.MakerOrderID,
		TakerOrderID:  message.TakerOrderID,
		UserID:        message.UserID,
		ProfileID:     message.ProfileID,
	}

	w.channel.order <- orderMessage
}

func (w *CoinbaseProWatcher) subscribeHandler() error {
	var (
		subscribeMessage       coinbasepro.Message
		subscribeMessageSigned coinbasepro.SignedMessage
		messageChannels        []coinbasepro.MessageChannel
		err                    error
	)

	messageChannels = []coinbasepro.MessageChannel{
		{
			Name:       ChannelTypeUser,
			ProductIds: w.updater.GetProductIDs(),
		},
	}

	subscribeMessage = coinbasepro.Message{
		Type:     MessageTypeSubscribe,
		Channels: messageChannels,
	}

	subscribeMessageSigned, err = subscribeMessage.Sign(w.userSettings.APISecret, w.userSettings.APIKey, w.userSettings.APIPassphrase)
	if utils.HasError(err) {
		logger.LogError(err)
		return nil
	}

	err = w.ws.WriteJSON(subscribeMessageSigned)
	if utils.HasError(err) {
		logger.LogError(err)
		return nil
	}

	// Start receiving messages within a separate go-routine
	go func() {
		for {
			var message coinbasepro.Message
			err = w.ws.ReadJSON(&message)
			if utils.HasError(err) {
				logger.LogError(err)
				return
			}
			w.handleWebSocketMessage(message)
		}
	}()

	return nil
}
