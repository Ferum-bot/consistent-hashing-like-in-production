package metrics

import (
	"context"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/wallet/internal/models"
)

type StatsService interface {
	Stats(ctx context.Context) (models.Stats, error)
}

type TopologyFence interface {
	Current() int64
	Strict() bool
}
