package client

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

type TradeTicker struct {
	TradeID   int
	ProductID string
	Time      time.Time
	Side      string
	Price     decimal.Decimal
	Size      decimal.Decimal
	Ask       decimal.Decimal
	Bid       decimal.Decimal
	Spread    decimal.Decimal // Spread = Ask - Bid
}

func (t TradeTicker) String() string {
	return fmt.Sprintf(
		"Ticker[%s]: Side:%s, Price:%s, Size:%s Ask:%s, Bid:%s, Spread:%s",
		t.ProductID,
		t.Side,
		t.Price,
		t.Size,
		t.Ask,
		t.Bid,
		t.Spread,
	)
}

type MovingAverage struct {
	sum     decimal.Decimal
	n       decimal.Decimal
	average decimal.Decimal
}

// Add adds a new value to the moving average
func (ma *MovingAverage) Add(value decimal.Decimal) {
	ma.sum = ma.sum.Add(value)
	ma.n = ma.n.Add(decimal.NewFromInt(1))
	ma.average = ma.sum.Div(ma.n)
}

// Get returns the current moving average
func (ma *MovingAverage) Get() decimal.Decimal {
	return ma.average
}

func (ma MovingAverage) String() string {
	return fmt.Sprintf("Current average: %s based on %s values", ma.average.StringFixed(2), ma.n)
}

type Spread struct {
	CurrentValue decimal.Decimal
	LastValue    decimal.Decimal
	Min, Max     decimal.Decimal
	Average      MovingAverage
}

// Set sets the current spread value
func (s *Spread) Set(currentValue decimal.Decimal) {
	var spread decimal.Decimal
	if s.LastValue.GreaterThan(decimal.Zero) {
		spread = s.LastValue.Sub(currentValue).Abs()
		if spread.IsZero() {
			return
		}
		s.Average.Add(spread)
	}
	if s.Min.IsZero() || spread.LessThan(s.Min) {
		s.Min = spread
	}
	if spread.GreaterThan(s.Max) {
		s.Max = spread
	}
	s.LastValue = s.CurrentValue
	s.CurrentValue = currentValue
}

// HasChanged returns true if the current and the last value are different
func (s *Spread) HasChanged() bool {
	if s.CurrentValue.Equal(s.LastValue) {
		return false
	}
	if s.CurrentValue.IsZero() && s.LastValue.IsZero() {
		return false
	}
	return true
}

func (s Spread) String() string {
	return fmt.Sprintf("Spread: %s (min), %s (max), %s (average)", s.Min, s.Max, s.Average)
}
