package main

import (
	"github.com/sknr/go-coinbasepro-notifier/internal/app"
	"github.com/sknr/go-coinbasepro-notifier/internal/telegram"
)

func main() {
	// Send a push message to the admin in case the app panicked
	defer telegram.SendAdminPushMessageWhenPanic()
	a := app.New()
	a.Start()
}
