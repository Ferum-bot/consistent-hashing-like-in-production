package services

import (
	"context"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/wallet/internal/models"
)

type Repository interface {
	Get(ctx context.Context, id int64) (models.User, error)
	Update(ctx context.Context, id int64, fn func(*models.User) error) error
	Save(ctx context.Context, users []models.User) error
	DeleteMany(ctx context.Context, ids []int64) (int, error)
	Snapshot(ctx context.Context) ([]models.User, error)
	Stats(ctx context.Context) (models.Stats, error)
}
