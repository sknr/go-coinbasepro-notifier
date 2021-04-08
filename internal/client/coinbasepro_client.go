package client

import (
	"fmt"
	ws "github.com/gorilla/websocket"
	. "github.com/sknr/go-coinbasepro-notifier/internal"
	"github.com/sknr/go-coinbasepro-notifier/internal/database"
	"github.com/sknr/go-coinbasepro-notifier/internal/logger"
	"github.com/sknr/go-coinbasepro-notifier/internal/telegram"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/preichenberger/go-coinbasepro/v2"
	"github.com/shopspring/decimal"
)

const (
	// Coinbase Pro channel types
	ChannelTypeFull      = "full"
	ChannelTypeHeartbeat = "heartbeat"
	ChannelTypeMatches   = "matches"
	ChannelTypeStatus    = "status"
	ChannelTypeTicker    = "ticker"
	ChannelTypeUser      = "user"

	// Coinbase Pro message types
	MessageTypeActivate      = "activate"
	MessageTypeChange        = "change"
	MessageTypeDone          = "done"
	MessageTypeError         = "error"
	MessageTypeHeartbeat     = "heartbeat"
	MessageTypeMatch         = "match"
	MessageTypeOpen          = "open"
	MessageTypeReceived      = "received"
	MessageTypeStatus        = "status"
	MessageTypeSubscriptions = "subscriptions"
	MessageTypeTicker        = "ticker"

	// Coinbase Pro order reasons
	OrderReasonFilled   = "filled"
	OrderReasonCanceled = "canceled"

	// internal const
	coinbaseProURL          = "https://api.pro.coinbase.com"
	coinbaseProWebSocketURL = "wss://ws-feed.pro.coinbase.com"
	webSocketRetryCount     = 10

	webSocketWriteWait = 3 * time.Second
	webSocketPingWait  = 45 * time.Second
	webSocketPongWait  = 60 * time.Second
)

type CoinbaseProClient struct {
	coinbasepro.Client
	retries      int                   // Current websocket connection retries
	userSettings database.UserSettings // Current user settings
	channel      channel               // Communication channels
	terminate    chan os.Signal        // Channel for terminating the app via os.Interrupt signal
}

type CoinbaseProClientConfig struct {
	coinbasepro.ClientConfig
}

type channel struct {
	order  chan OrderMessage
	ticker chan TradeTicker
	close  chan struct{}
	error  chan error
}

// NewCoinbaseProClient creates a new Coinbase Pro client
func NewCoinbaseProClient(userSettings database.UserSettings, config *CoinbaseProClientConfig) *CoinbaseProClient {
	baseURL := config.BaseURL

	if baseURL == "" {
		baseURL = coinbaseProURL
	}

	client := CoinbaseProClient{
		Client: coinbasepro.Client{
			BaseURL:    baseURL,
			Key:        config.Key,
			Passphrase: config.Passphrase,
			Secret:     config.Secret,
			HTTPClient: &http.Client{
				Timeout: 25 * time.Second,
			},
			RetryCount: 0,
		},
	}

	// Set the telegram ID of the current user
	client.userSettings = userSettings

	// Initialize channels
	client.channel.ticker = make(chan TradeTicker, 5)
	client.channel.order = make(chan OrderMessage, 5)
	client.channel.error = make(chan error, 1)
	client.channel.close = make(chan struct{}, 1)

	// Create os interrupt channel for user cancellation
	client.terminate = make(chan os.Signal, 1)
	// Capture the interrupt signal for app termination handling
	signal.Notify(client.terminate, syscall.SIGINT, syscall.SIGTERM)

	return &client
}

// UpdateConfig updates the coinbasepro client config
func (c *CoinbaseProClient) UpdateConfig(config CoinbaseProClientConfig) {
	if config.BaseURL != "" {
		c.BaseURL = config.BaseURL
	}
	if config.Key != "" {
		c.Key = config.Key
	}
	if config.Passphrase != "" {
		c.Passphrase = config.Passphrase
	}
	if config.Secret != "" {
		c.Secret = config.Secret
	}
}

// Subscribe creates a new WebSocket and subscribes to the given message channels. Receiving and handling of messages
// is handled within a separate go-routine to be non-blocking.
func (c *CoinbaseProClient) Subscribe(conn *ws.Conn, messageChannel []coinbasepro.MessageChannel) error {
	var (
		subscribeMessage       coinbasepro.Message
		subscribeMessageSigned coinbasepro.SignedMessage
		err                    error
	)
	subscribeMessage = coinbasepro.Message{
		Type:     "subscribe",
		Channels: messageChannel,
	}

	subscribeMessageSigned, err = subscribeMessage.Sign(c.Secret, c.Key, c.Passphrase)
	if HasError(err) {
		return err
	}

	err = conn.WriteJSON(subscribeMessageSigned)
	if HasError(err) {
		return err
	}

	// Start receiving messages within a separate go-routine
	go func() {
		var message coinbasepro.Message
		for {
			err = conn.ReadJSON(&message)
			if HasError(err) {
				// Send an error through the channel in order to automatically reconnect
				c.channel.error <- err
				return
			}
			c.handleWebSocketMessage(message)
		}
	}()

	return nil
}

// Watch starts and subscribes the Coinbase Pro Notifier for the given messageChannel
func (c *CoinbaseProClient) Watch(messageChannel []coinbasepro.MessageChannel) {
	var (
		dialer     ws.Dialer
		pingTicker = time.NewTicker(webSocketPingWait)
	)

	webSocketURL := os.Getenv("COINBASE_PRO_WEBSOCKET_URL")
	if webSocketURL == "" {
		webSocketURL = coinbaseProWebSocketURL
	}

	// Create new WebSocket connection
	wsConn, _, err := dialer.Dial(webSocketURL, nil)
	if HasError(err) {
		logger.LogError(err)
		return
	}

	defer func() {
		logger.LogInfof("Closing the client with TelegramID: %s", c.userSettings.TelegramID)
		logger.LogErrorIfExists(wsConn.Close())
	}()

	// Set read deadline for ping / pong handling
	_ = wsConn.SetReadDeadline(time.Now().Add(webSocketPongWait))
	wsConn.SetPongHandler(func(string) error { _ = wsConn.SetReadDeadline(time.Now().Add(webSocketPongWait)); return nil })

	// Try to subscribe to web message channel
	err = c.Subscribe(wsConn, messageChannel)
	if err != nil {
		logger.LogError(err)
		return
	}
	// Reset retries if connection was successful
	c.retries = 0

	// Block until app termination (CTRL-C) or we receive an message on the close channel
	for {
		select {
		case <-pingTicker.C:
			_ = wsConn.SetWriteDeadline(time.Now().Add(webSocketWriteWait))
			_ = wsConn.WriteMessage(ws.PingMessage, []byte{})
		case <-c.terminate:
			//SendPushMessage("Coinbase Pro Notifier has been shut down")
			return
		case wsErr := <-c.channel.error:
			logger.LogError(wsErr)
			if c.retries == webSocketRetryCount {
				logger.LogWarn("Max websocket connection retries reached! -> Stopping client")
				telegram.SendAdminPushMessage(fmt.Sprintf("Max websocket connection retries reached!\nStopping client with ID %s. Manual intervention required for restart",c.userSettings.TelegramID))
				return
			}
			time.Sleep(time.Second * (1 << c.retries))
			c.retries++
			// Restart websocket due to abnormal closing
			go c.Watch(messageChannel)
			return
		case <-c.channel.close:
			return
		//case tickerMessage := <-c.channel.ticker:
		case orderMessage := <-c.channel.order:
			telegram.SendPushMessage(c.userSettings.TelegramID, orderMessage.String())
		}
	}
}

func (c *CoinbaseProClient) Close() {
	c.channel.close <- struct{}{}
}

/********************
 * internal methods *
 ********************/

// handleWebSocketMessage handles the received WebSocket coinbasepro.Message
func (c *CoinbaseProClient) handleWebSocketMessage(message coinbasepro.Message) {
	switch message.Type {
	case MessageTypeActivate, MessageTypeChange, MessageTypeDone, MessageTypeMatch, MessageTypeOpen, MessageTypeReceived:
		logger.LogInfo("Order-Message", c.userSettings.TelegramID, message)
		c.handleOrderMessage(message)
	case MessageTypeError:
		logger.LogWarn("ErrorMessage", c.userSettings.TelegramID, message.Message)
		if message.Message == "Authentication Failed" {
			telegram.SendPushMessage(c.userSettings.TelegramID, "Coinbase Pro authentication failed. Please check your API-Settings, in order to get informed about your order changes.")
		}
		telegram.SendAdminPushMessage(fmt.Sprintf("Received an error message for user %s (%s)\nErrorMessage: %s", c.userSettings.FirstName, c.userSettings.TelegramID, message.Message))
	case MessageTypeSubscriptions:
		logger.LogInfo("Successfully subscribed to channels", c.userSettings.TelegramID, message.Channels)
	case MessageTypeTicker:
		logger.LogInfo("Ticker-Message", c.userSettings.TelegramID, message)
		c.handleTickerMessage(message)
	case MessageTypeStatus:
		logger.LogInfo("Status-Message", c.userSettings.TelegramID, message)
	default:
		logger.LogInfof("Received message of unknown type %q", message.Type)
		logger.LogInfof("Message: %#v", message)
	}
}

// handleTickerMessage converts a coinbasepro.Message into a TradeTicker
func (c *CoinbaseProClient) handleTickerMessage(message coinbasepro.Message) {
	var ask, bid decimal.Decimal
	ask = StringToDecimal(message.BestAsk)
	bid = StringToDecimal(message.BestBid)

	ticker := TradeTicker{
		TradeID:   message.TradeID,
		ProductID: message.ProductID,
		Time:      message.Time.Time(),
		Side:      message.Side,
		Price:     StringToDecimal(message.Price),
		Size:      StringToDecimal(message.LastSize),
		Ask:       ask,
		Bid:       bid,
		Spread:    ask.Sub(bid),
	}
	c.channel.ticker <- ticker
}

// handleOrderMessage converts a coinbasepro.Message into an OrderMessage
func (c *CoinbaseProClient) handleOrderMessage(message coinbasepro.Message) {
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

	c.channel.order <- orderMessage
}

// GetAllAvailableProductIDs returns all available product ids from Coinbase Pro
func (c *CoinbaseProClient) GetAllAvailableProductIDs() ([]string, error) {
	products, err := c.GetProducts()
	if HasError(err) {
		logger.LogError(err)
		return nil, err
	}

	var result []string
	for _, product := range products {
		result = append(result, product.ID)
	}

	return result, nil
}
