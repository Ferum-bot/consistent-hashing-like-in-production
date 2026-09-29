package services

import (
	"context"
	"time"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/loadgen/internal/models"
)

type Ledger interface {
	Load(ctx context.Context, path string) error
	Users() int
	Wallets(userID int64) []models.Wallet
	BeginWrite(userID int64) models.WriteTicket
	Apply(ticket models.WriteTicket, currency string, delta int64)
	MarkUncertain(ticket models.WriteTicket)
	EndWrite(ticket models.WriteTicket)
	Observe(userID int64) models.Observation
	Unchanged(before models.Observation) bool
	Stats() models.LedgerStats
}

type RouterClient interface {
	Get(ctx context.Context, router string, userID int64) (models.UserResponse, error)
	Deposit(ctx context.Context, router string, userID int64, currency string, amount int64) (int, error)
	Withdraw(ctx context.Context, router string, userID int64, currency string, amount int64) (int, error)
}

type Observer interface {
	ObserveRequest(kind models.OperationKind, router string, status int, duration time.Duration)
	ObserveDropped()
	ObserveCheck(result models.CheckResult)
	ObserveProfile(profile models.Profile)
}
