package api

import (
	"context"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/loadgen/internal/models"
)

type GeneratorService interface {
	Profile() models.Profile
	UpdateProfile(change func(*models.Profile)) (models.Profile, error)
}

type LedgerService interface {
	Reload(ctx context.Context) (models.LedgerStats, error)
	Stats(ctx context.Context) (models.LedgerStats, error)
}
