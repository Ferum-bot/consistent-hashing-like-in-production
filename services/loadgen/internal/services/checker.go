package services

import (
	"math/rand/v2"
	"net/http"
	"sync/atomic"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/loadgen/internal/models"
)

const touchedCapacity = 4096

type Checker struct {
	ledger   Ledger
	observer Observer

	touched [touchedCapacity]atomic.Int64
	cursor  atomic.Uint64
}

func NewChecker(ledger Ledger, observer Observer) *Checker {
	return &Checker{ledger: ledger, observer: observer}
}

func (c *Checker) Touch(userID int64) {
	slot := c.cursor.Add(1) % touchedCapacity

	c.touched[slot].Store(userID)
}

func (c *Checker) RecentlyTouched() (int64, bool) {
	userID := c.touched[rand.IntN(touchedCapacity)].Load()

	return userID, userID != 0
}

func (c *Checker) Check(before models.Observation, response models.UserResponse) models.CheckResult {
	result := c.compare(before, response)

	c.observer.ObserveCheck(result)

	return result
}

func (c *Checker) compare(before models.Observation, response models.UserResponse) models.CheckResult {
	if before.Busy {
		return models.CheckSkipped
	}

	if response.Status != http.StatusOK && response.Status != http.StatusNotFound {
		return models.CheckSkipped
	}

	expected := c.ledger.Wallets(before.UserID)

	if !c.ledger.Unchanged(before) {
		return models.CheckSkipped
	}

	if response.Status == http.StatusNotFound {
		return models.CheckMissing
	}

	return compareWallets(expected, response.Wallets)
}

func compareWallets(expected, actual []models.Wallet) models.CheckResult {
	balances := make(map[string]int64, len(actual))

	for _, wallet := range actual {
		balances[wallet.Currency] = wallet.Balance
	}

	lost, extra := false, false

	for _, wallet := range expected {
		got := balances[wallet.Currency]

		if got < wallet.Balance {
			lost = true
		}

		if got > wallet.Balance {
			extra = true
		}
	}

	switch {
	case lost:
		return models.CheckLost
	case extra:
		return models.CheckExtra
	default:
		return models.CheckOK
	}
}
