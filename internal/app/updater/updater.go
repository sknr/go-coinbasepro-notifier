package updater

import (
	"github.com/preichenberger/go-coinbasepro/v2"
	"github.com/sknr/go-coinbasepro-notifier/internal/logger"
	"github.com/sknr/go-coinbasepro-notifier/internal/utils"
	"sync"
	"time"
)

type Updater struct {
	client     *coinbasepro.Client
	ticker     *time.Ticker
	productIDs []string
	mu         sync.RWMutex
}

func New() *Updater {
	u := &Updater{
		client: coinbasepro.NewClient(),
	}
	u.productIDs = u.getProductIDsFromCoinbase()
	// Start background task for updating
	go u.Update()
	return u
}

func (u *Updater) Update() {
	if u.ticker == nil {
		u.ticker = time.NewTicker(6 * time.Hour)
	}
	for range u.ticker.C {
		u.mu.Lock()
		u.productIDs = u.getProductIDsFromCoinbase()
		u.mu.Unlock()
	}
}

func (u *Updater) Stop() {
	u.ticker.Stop()
	u.ticker = nil
}

func (u *Updater) GetProductIDs() []string {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.productIDs
}

func (u *Updater) getProductIDsFromCoinbase() []string {
	products, err := u.client.GetProducts()
	if utils.HasError(err) {
		logger.LogError(err)
		return []string{}
	}
	var productIDs []string
	for _, product := range products {
		productIDs = append(productIDs, product.ID)
	}
	return productIDs
}
