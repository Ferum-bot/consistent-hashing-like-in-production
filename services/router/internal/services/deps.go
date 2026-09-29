package services

import (
	"context"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/sharding"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/router/internal/models"
)

type TopologyObserver interface {
	ObserveTopology(stats models.TopologyStats)
}

type MigrationObserver interface {
	ObserveMigrationActive(active bool)
	ObserveCopied(users int)
}

type GateObserver interface {
	ObserveGateWaiting(delta int)
	ObserveGateRejected()
}

type WalletClient interface {
	Import(ctx context.Context, node sharding.Node, version int64, body []byte) error
	Export(ctx context.Context, node sharding.Node, version int64, visit func(line []byte) error) error
	Evict(ctx context.Context, node sharding.Node, version int64, userIDs []int64) error
}

type Fencer interface {
	Fence(ctx context.Context, node sharding.Node, version int64, strict bool) error
}
