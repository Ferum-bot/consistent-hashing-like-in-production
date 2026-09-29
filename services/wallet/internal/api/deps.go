package api

import (
	"context"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/wallet/internal/models"
)

type WalletService interface {
	User(ctx context.Context, id int64) (models.User, error)
	Deposit(ctx context.Context, userID int64, currency string, amount int64) (models.Wallet, error)
	Withdraw(ctx context.Context, userID int64, currency string, amount int64) (models.Wallet, error)
}

type MigrationService interface {
	Export(ctx context.Context) ([]models.User, error)
	Import(ctx context.Context, users []models.User) error
	Evict(ctx context.Context, ids []int64) (int, error)
}

type StatsService interface {
	Stats(ctx context.Context) (models.Stats, error)
}

type TopologyFence interface {
	Current() int64
	Strict() bool
	SetStrict(strict bool)
}
