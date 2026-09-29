package services

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/sharding"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/router/internal/models"
)

type routingState struct {
	topology models.Topology
	strategy sharding.Strategy
}

type pendingChange struct {
	next *routingState
	done chan struct{}
}

type routingView struct {
	current *routingState
	pending *pendingChange
	fence   int64
}

func (v routingView) moves(userID int64) bool {
	return v.pending != nil && moves(v.current, v.pending.next, userID)
}

func (v routingView) route(userID int64) models.Route {
	route := locate(v.current, userID)
	route.FenceVersion = v.fence

	return route
}

type TopologyService struct {
	changes sync.Mutex
	current atomic.Pointer[routingState]
	pending atomic.Pointer[pendingChange]
	writes  sync.RWMutex
	newest  atomic.Int64

	name          string
	fencer        Fencer
	deferred      chan deferredChange
	queued        atomic.Int64
	catchUpAt     atomic.Int64
	deferredError atomic.Pointer[string]

	observer          TopologyObserver
	migrationObserver MigrationObserver
	keySpace          int64
}

func NewTopologyService(
	name string,
	keySpace int64,
	fencer Fencer,
	observer TopologyObserver,
	migrationObserver MigrationObserver,
) *TopologyService {
	return &TopologyService{
		name:              name,
		keySpace:          keySpace,
		fencer:            fencer,
		observer:          observer,
		migrationObserver: migrationObserver,
		deferred:          make(chan deferredChange, deferredQueueSize),
	}
}

func (s *TopologyService) Apply(ctx context.Context, topology models.Topology) (time.Duration, error) {
	next, err := newRoutingState(topology)
	if err != nil {
		return 0, err
	}

	if delay := s.delayFor(next); delay > 0 {
		return s.postpone(delay, "apply", func(ctx context.Context) error {
			return s.applyNow(ctx, next)
		})
	}

	return 0, s.applyNow(ctx, next)
}

func (s *TopologyService) Prepare(ctx context.Context, topology models.Topology) (time.Duration, error) {
	next, err := newRoutingState(topology)
	if err != nil {
		return 0, err
	}

	if delay := s.delayFor(next); delay > 0 {
		return s.postpone(delay, "prepare", func(ctx context.Context) error {
			return s.prepareNow(ctx, next)
		})
	}

	return 0, s.prepareNow(ctx, next)
}

func (s *TopologyService) Commit(ctx context.Context, version int64) (time.Duration, error) {
	if delay := s.currentLag(); delay > 0 {
		return s.postpone(delay, "commit", func(ctx context.Context) error {
			return s.commitNow(ctx, version)
		})
	}

	return 0, s.commitNow(ctx, version)
}

func (s *TopologyService) Abort(ctx context.Context) (time.Duration, error) {
	if delay := s.currentLag(); delay > 0 {
		return s.postpone(delay, "abort", s.abortNow)
	}

	return 0, s.abortNow(ctx)
}

func (s *TopologyService) applyNow(ctx context.Context, next *routingState) error {
	s.changes.Lock()
	defer s.changes.Unlock()

	if change := s.pending.Load(); change != nil {
		return models.NewErrMigrationInProgress(change.next.topology.Version)
	}

	previous := s.current.Load()

	if previous != nil && next.topology.Version <= previous.topology.Version {
		return models.NewErrStaleTopology(previous.topology.Version, next.topology.Version)
	}

	if err := s.announce(ctx, max(next.topology.Version, s.newest.Load()), next, previous); err != nil {
		return err
	}

	s.remember(next.topology.Version)
	s.current.Store(next)
	s.observeTopology(next, previous)
	s.clearDeferredError()

	return nil
}

func (s *TopologyService) prepareNow(ctx context.Context, next *routingState) error {
	s.changes.Lock()
	defer s.changes.Unlock()

	current := s.current.Load()
	if current == nil {
		return models.ErrNoTopology
	}

	if change := s.pending.Load(); change != nil {
		return models.NewErrMigrationInProgress(change.next.topology.Version)
	}

	if next.topology.Version <= current.topology.Version {
		return models.NewErrStaleTopology(current.topology.Version, next.topology.Version)
	}

	s.remember(next.topology.Version)

	s.writes.Lock()
	s.pending.Store(&pendingChange{next: next, done: make(chan struct{})})
	s.writes.Unlock()

	s.migrationObserver.ObserveMigrationActive(true)

	if err := s.announce(ctx, s.newest.Load(), next, current); err != nil {
		if abortErr := s.abortLocked(ctx); abortErr != nil {
			slog.Error("roll back after a failed announce", "router", s.name, "version", next.topology.Version, "err", abortErr)
		}

		return err
	}

	s.clearDeferredError()

	return nil
}

func (s *TopologyService) commitNow(_ context.Context, version int64) error {
	s.changes.Lock()
	defer s.changes.Unlock()

	change := s.pending.Load()
	if change == nil {
		return models.ErrNoMigration
	}

	if change.next.topology.Version != version {
		return models.NewErrPendingVersion(change.next.topology.Version, version)
	}

	s.writes.Lock()
	previous := s.current.Swap(change.next)
	s.pending.Store(nil)
	s.writes.Unlock()

	close(change.done)

	s.migrationObserver.ObserveMigrationActive(false)
	s.observeTopology(change.next, previous)
	s.clearDeferredError()

	return nil
}

func (s *TopologyService) abortNow(ctx context.Context) error {
	s.changes.Lock()
	defer s.changes.Unlock()

	return s.abortLocked(ctx)
}

func (s *TopologyService) abortLocked(ctx context.Context) error {
	change := s.pending.Swap(nil)
	if change == nil {
		return models.ErrNoMigration
	}

	close(change.done)

	s.migrationObserver.ObserveMigrationActive(false)

	return s.announce(ctx, s.newest.Load(), s.current.Load(), change.next)
}

func (s *TopologyService) Current(context.Context) (models.Topology, error) {
	view, err := s.view()
	if err != nil {
		return models.Topology{}, err
	}

	return view.current.topology, nil
}

func (s *TopologyService) State(context.Context) (models.RouterState, error) {
	view, err := s.view()
	if err != nil {
		return models.RouterState{}, err
	}

	state := models.RouterState{Topology: view.current.topology}

	if view.pending != nil {
		state.PendingVersion = view.pending.next.topology.Version
	}

	if failure := s.deferredError.Load(); failure != nil {
		state.DeferredError = *failure
	}

	return state, nil
}

func (s *TopologyService) view() (routingView, error) {
	pending := s.pending.Load()

	current := s.current.Load()
	if current == nil {
		return routingView{}, models.ErrNoTopology
	}

	return routingView{current: current, pending: pending, fence: s.fenceVersion(current, pending)}, nil
}

func (s *TopologyService) writeView() (routingView, func(), error) {
	s.writes.RLock()

	view, err := s.view()
	if err != nil {
		s.writes.RUnlock()
		return routingView{}, nil, err
	}

	return view, s.writes.RUnlock, nil
}

func (s *TopologyService) stillPending(change *pendingChange) bool {
	return s.pending.Load() == change
}
