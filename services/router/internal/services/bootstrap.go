package services

import (
	"bytes"
	"context"
	"os"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/snapshot"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/router/internal/models"
)

type BootstrapService struct {
	topology     *TopologyService
	wallets      WalletClient
	snapshotPath string
}

func NewBootstrapService(topology *TopologyService, wallets WalletClient, snapshotPath string) *BootstrapService {
	return &BootstrapService{topology: topology, wallets: wallets, snapshotPath: snapshotPath}
}

func (s *BootstrapService) Run(ctx context.Context) (models.BootstrapResult, error) {
	view, err := s.topology.view()
	if err != nil {
		return models.BootstrapResult{}, err
	}

	file, err := os.Open(s.snapshotPath)
	if err != nil {
		return models.BootstrapResult{}, models.NewErrReadSnapshot(err)
	}
	defer file.Close()

	scanner := snapshot.NewScanner(file)

	result := models.BootstrapResult{PerNode: map[string]int{}}
	batches := newBatcher(s.wallets, view.fence)

	for line := 1; scanner.Scan(); line++ {
		raw := bytes.TrimSpace(scanner.Bytes())
		if len(raw) == 0 {
			continue
		}

		userID, err := snapshot.UserID(raw)
		if err != nil {
			return result, models.NewErrSnapshotLine(line, err)
		}

		node := view.route(userID).Node

		if err := batches.add(ctx, node, raw); err != nil {
			return result, err
		}

		result.Users++
		result.PerNode[node.ID]++
	}

	if err := scanner.Err(); err != nil {
		return result, models.NewErrReadSnapshot(err)
	}

	if err := batches.flushAll(ctx); err != nil {
		return result, err
	}

	return result, nil
}
