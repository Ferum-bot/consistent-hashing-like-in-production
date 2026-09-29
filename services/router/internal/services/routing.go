package services

import (
	"context"
	"strconv"
	"time"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/sharding"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/router/internal/models"
)

type RoutingService struct {
	topology    *TopologyService
	observer    GateObserver
	gateTimeout time.Duration
}

func NewRoutingService(topology *TopologyService, observer GateObserver, gateTimeout time.Duration) *RoutingService {
	return &RoutingService{topology: topology, observer: observer, gateTimeout: gateTimeout}
}

func (s *RoutingService) Locate(_ context.Context, userID int64) (models.Route, error) {
	view, err := s.topology.view()
	if err != nil {
		return models.Route{}, err
	}

	return view.route(userID), nil
}

func (s *RoutingService) LocateWrite(ctx context.Context, userID int64) (models.Route, func(), error) {
	timeout := time.NewTimer(s.gateTimeout)
	defer timeout.Stop()

	for {
		view, release, err := s.topology.writeView()
		if err != nil {
			return models.Route{}, nil, err
		}

		if !view.moves(userID) {
			return view.route(userID), release, nil
		}

		release()

		if err := s.wait(ctx, view.pending, timeout, userID); err != nil {
			return models.Route{}, nil, err
		}
	}
}

func (s *RoutingService) wait(ctx context.Context, change *pendingChange, timeout *time.Timer, userID int64) error {
	s.observer.ObserveGateWaiting(1)
	defer s.observer.ObserveGateWaiting(-1)

	select {
	case <-change.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-timeout.C:
		s.observer.ObserveGateRejected()

		return models.NewErrMigrationTimeout(userID)
	}
}

func moves(current, next *routingState, userID int64) bool {
	key := strconv.FormatInt(userID, 10)

	return current.strategy.Locate(key).ID != next.strategy.Locate(key).ID
}

type ringStrategy interface {
	Point(key string) sharding.RingPoint
}

func locate(state *routingState, userID int64) models.Route {
	key := strconv.FormatInt(userID, 10)

	route := models.Route{
		UserID:   userID,
		Key:      key,
		Hash:     sharding.Hash(key),
		Node:     state.strategy.Locate(key),
		Version:  state.topology.Version,
		Strategy: state.topology.Strategy,
	}

	if ring, ok := state.strategy.(ringStrategy); ok {
		point := ring.Point(key)
		route.Point = &models.RingPoint{Node: point.Node.ID, Index: point.Index, Hash: point.Hash}
	}

	return route
}
