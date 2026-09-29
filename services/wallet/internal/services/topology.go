package services

import (
	"sync/atomic"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/wallet/internal/models"
)

type TopologyFence struct {
	version atomic.Int64
	strict  atomic.Bool
}

func NewTopologyFence() *TopologyFence {
	return &TopologyFence{}
}

func (f *TopologyFence) Observe(version int64) (int64, error) {
	for {
		current := f.version.Load()

		if version < current {
			return current, models.NewErrStaleVersion(current, version)
		}

		if version == current || f.version.CompareAndSwap(current, version) {
			return version, nil
		}
	}
}

func (f *TopologyFence) Current() int64 {
	return f.version.Load()
}

func (f *TopologyFence) SetStrict(strict bool) {
	f.strict.Store(strict)
}

func (f *TopologyFence) Strict() bool {
	return f.strict.Load()
}
