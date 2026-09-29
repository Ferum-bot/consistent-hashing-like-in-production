package services

import (
	"context"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/loadgen/internal/models"
)

type LedgerService struct {
	ledger       Ledger
	snapshotPath string
}

func NewLedgerService(ledger Ledger, snapshotPath string) *LedgerService {
	return &LedgerService{ledger: ledger, snapshotPath: snapshotPath}
}

func (s *LedgerService) Reload(ctx context.Context) (models.LedgerStats, error) {
	if err := s.ledger.Load(ctx, s.snapshotPath); err != nil {
		return models.LedgerStats{}, err
	}

	return s.ledger.Stats(), nil
}

func (s *LedgerService) Stats(_ context.Context) (models.LedgerStats, error) {
	stats := s.ledger.Stats()

	if !stats.Loaded {
		return stats, models.ErrLedgerNotLoaded
	}

	return stats, nil
}
