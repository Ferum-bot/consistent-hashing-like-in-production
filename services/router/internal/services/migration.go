package services

import (
	"context"
	"strconv"
	"sync"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/sharding"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/snapshot"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/router/internal/models"
)

type copiedUsers struct {
	source sharding.Node
	ids    []int64
}

type MigrationService struct {
	topology *TopologyService
	wallets  WalletClient
	observer MigrationObserver

	mu      sync.Mutex
	version int64
	copied  map[string]*copiedUsers
}

func NewMigrationService(topology *TopologyService, wallets WalletClient, observer MigrationObserver) *MigrationService {
	return &MigrationService{topology: topology, wallets: wallets, observer: observer}
}

func (s *MigrationService) Run(ctx context.Context) (models.MigrationResult, error) {
	view, err := s.topology.view()
	if err != nil {
		return models.MigrationResult{}, err
	}

	change := view.pending
	if change == nil {
		return models.MigrationResult{}, models.ErrNoMigration
	}

	current := view.current
	version := view.fence
	result := models.MigrationResult{PerNode: map[string]int{}}
	copied := map[string]*copiedUsers{}
	batches := newBatcher(s.wallets, version)

	for _, source := range current.topology.Nodes {
		copied[source.ID] = &copiedUsers{source: source}

		err := s.wallets.Export(ctx, source, version, func(line []byte) error {
			userID, err := snapshot.UserID(line)
			if err != nil {
				return err
			}

			key := strconv.FormatInt(userID, 10)

			if current.strategy.Locate(key).ID != source.ID {
				return nil
			}

			target := change.next.strategy.Locate(key)

			if target.ID == source.ID {
				return nil
			}

			if err := batches.add(ctx, target, line); err != nil {
				return err
			}

			copied[source.ID].ids = append(copied[source.ID].ids, userID)
			result.Users++
			result.PerNode[target.ID]++

			return nil
		})
		if err != nil {
			return result, err
		}
	}

	if err := batches.flushAll(ctx); err != nil {
		return result, err
	}

	if !s.topology.stillPending(change) {
		return result, models.ErrNoMigration
	}

	s.mu.Lock()
	s.version = change.next.topology.Version
	s.copied = copied
	s.mu.Unlock()

	s.observer.ObserveCopied(result.Users)

	return result, nil
}

func (s *MigrationService) Cleanup(ctx context.Context) (models.CleanupResult, error) {
	view, err := s.topology.view()
	if err != nil {
		return models.CleanupResult{}, err
	}

	if view.pending != nil {
		return models.CleanupResult{}, models.NewErrMigrationInProgress(view.pending.next.topology.Version)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	result := models.CleanupResult{}

	if s.version != view.current.topology.Version {
		s.copied = nil
		return result, nil
	}

	for _, users := range s.copied {
		for len(users.ids) > 0 {
			chunk := users.ids[:min(usersPerRequest, len(users.ids))]

			if err := s.wallets.Evict(ctx, users.source, view.fence, chunk); err != nil {
				return result, err
			}

			users.ids = users.ids[len(chunk):]
			result.Users += len(chunk)
		}
	}

	s.copied = nil

	return result, nil
}
