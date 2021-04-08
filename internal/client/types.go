package client

import (
	"fmt"
	"github.com/sknr/go-coinbasepro-notifier/internal/logger"
	"time"
)

type OrderMessage struct {
	Type          string
	Time          *time.Time
	ProductID     string
	Sequence      int64
	OrderID       string
	Funds         string
	NewFunds      string
	OldFunds      string
	Side          string
	OrderType     string
	Price         string
	RemainingSize string
	Size          string
	NewSize       string
	OldSize       string
	Reason        string
	TradeID       int
	MakerOrderID  string
	TakerOrderID  string
	UserID        string
	ProfileID     string
}

// String defines how the OrderMessage gets displayed
func (om OrderMessage) String() string {
	message := ""
	switch om.Type {
	case MessageTypeOpen:
		message = fmt.Sprintf("Order was successfully placed!\nTime: %s\nSide: %s\nOrderID: %s\nOrderType: %s\nProductID: %s\nSize: %s\nPrice: %s", om.Time.Format(time.RFC822), om.Side, om.OrderID, om.OrderType, om.ProductID, om.RemainingSize, om.Price)
	case MessageTypeDone:
		switch om.Reason {
		case OrderReasonFilled:
			if om.RemainingSize == "0" {
				message = fmt.Sprintf("Order was filled!\nTime: %s\nSide: %s\nOrderID: %s\nOrderType: %s\nProduct ID: %s\nPrice: %s", om.Time.Format(time.RFC822), om.Side, om.OrderID, om.OrderType, om.ProductID, om.Price)
			} else {
				message = fmt.Sprintf("Order was partially filled!\nTime: %s\nSide: %s\nOrderID: %s\nOrderType: %s\nProductID: %s\nRemaining Size: %s\nPrice: %s", om.Time.Format(time.RFC822), om.Side, om.OrderID, om.OrderType, om.ProductID, om.RemainingSize, om.Price)
			}
		case OrderReasonCanceled:
			message = fmt.Sprintf("Order was canceled!\nTime: %s\nSide: %s\nOrderID: %s\nProductID: %s\nSize: %s\nPrice: %s", om.Time.Format(time.RFC822),om.Side, om.OrderID, om.ProductID, om.RemainingSize, om.Price)
		default:
			logger.LogInfo("Unknown reason: %s", om.Reason)
		}
	}

	return message
}
