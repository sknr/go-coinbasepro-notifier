package main

import (
	"github.com/preichenberger/go-coinbasepro/v2"
	"github.com/sknr/go-coinbasepro-notifier/internal/client"
	"github.com/sknr/go-coinbasepro-notifier/internal/database"
	"os"
)

func main() {
	// See https://docs.pro.coinbase.com/#subscribe for more details
	channels := []coinbasepro.MessageChannel{
		{
			Name: client.ChannelTypeUser,
			ProductIds: []string{
				"BTC-EUR", "ETH-EUR", "ALGO-EUR", "NU-EUR", "ZRX-EUR",
			},
		},
	}

	config := client.CoinbaseProClientConfig{
		ClientConfig: coinbasepro.ClientConfig{
			BaseURL:    os.Getenv("COINBASE_PRO_BASEURL"),
			Key:        os.Getenv("COINBASE_PRO_KEY"),
			Passphrase: os.Getenv("COINBASE_PRO_PASSPHRASE"),
			Secret:     os.Getenv("COINBASE_PRO_SECRET"),
		},
	}

	userSettings := database.UserSettings{
		TelegramID: os.Getenv("TELEGRAM_CHAT_ID"),
	}

	cbp := client.NewCoinbaseProClient(userSettings, &config)
	cbp.Watch(channels)
}
