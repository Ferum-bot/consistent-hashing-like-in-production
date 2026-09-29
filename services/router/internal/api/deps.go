package api

import (
	"context"
	"net/http"
	"time"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/sharding"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/router/internal/models"
)

type RoutingService interface {
	Locate(ctx context.Context, userID int64) (models.Route, error)
	LocateWrite(ctx context.Context, userID int64) (models.Route, func(), error)
}

type Proxy interface {
	Forward(w http.ResponseWriter, r *http.Request, node sharding.Node, version int64)
}

type TopologyService interface {
	Apply(ctx context.Context, topology models.Topology) (time.Duration, error)
	Prepare(ctx context.Context, topology models.Topology) (time.Duration, error)
	Commit(ctx context.Context, version int64) (time.Duration, error)
	Abort(ctx context.Context) (time.Duration, error)
	State(ctx context.Context) (models.RouterState, error)
}

type MigrationService interface {
	Run(ctx context.Context) (models.MigrationResult, error)
	Cleanup(ctx context.Context) (models.CleanupResult, error)
}

type BootstrapService interface {
	Run(ctx context.Context) (models.BootstrapResult, error)
}
