package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/wait"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/router/internal/models"
)

const deferredQueueSize = 16

type deferredChange struct {
	at   time.Time
	name string
	run  func(ctx context.Context) error
}

func (s *TopologyService) RunDeferred(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case change := <-s.deferred:
			if !wait.Until(ctx, change.at) {
				return
			}

			err := change.run(ctx)

			s.queued.Add(-1)

			if err != nil {
				failure := change.name + ": " + err.Error()
				s.deferredError.CompareAndSwap(nil, &failure)

				slog.Warn("deferred topology change failed", "router", s.name, "change", change.name, "err", err)

				continue
			}

			slog.Info("deferred topology change applied", "router", s.name, "change", change.name)
		}
	}
}

func (s *TopologyService) delayFor(next *routingState) time.Duration {
	return max(next.topology.MembershipDelay[s.name], s.currentLag())
}

func (s *TopologyService) currentLag() time.Duration {
	if s.queued.Load() == 0 {
		return 0
	}

	return max(time.Until(time.Unix(0, s.catchUpAt.Load())), time.Nanosecond)
}

func (s *TopologyService) postpone(delay time.Duration, name string, run func(ctx context.Context) error) (time.Duration, error) {
	at := time.Now().Add(delay)

	s.queued.Add(1)

	select {
	case s.deferred <- deferredChange{at: at, name: name, run: run}:
	default:
		s.queued.Add(-1)
		return 0, models.NewErrTooManyDeferred(deferredQueueSize)
	}

	s.catchUpAt.Store(at.UnixNano())
	s.clearDeferredError()

	slog.Info("topology change postponed", "router", s.name, "change", name, "delay", delay)

	return delay, nil
}

func (s *TopologyService) clearDeferredError() {
	s.deferredError.Store(nil)
}
