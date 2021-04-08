package database

import (
	"time"
)

type UserSettings struct {
	TelegramID    string `gorm:"primaryKey"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Username      string
	FirstName     string
	LastName      string
	PhotoURL      string
	APIKey        string
	APIPassphrase string
	APISecret     string
}
