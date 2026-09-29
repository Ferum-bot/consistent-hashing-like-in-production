package services

import (
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/sharding"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/router/internal/models"
)

func buildStrategy(topology models.Topology) (sharding.Strategy, error) {
	switch topology.Strategy {
	case models.StrategyMod:
		return sharding.NewMod(topology.Nodes), nil
	case models.StrategyRing:
		return sharding.NewRing(topology.Nodes, topology.VNodes), nil
	default:
		return nil, models.NewErrUnknownStrategy(topology.Strategy)
	}
}
