package services

import (
	"strconv"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/router/internal/models"
)

const keyShareSamples = 100_000

func (s *TopologyService) observeTopology(next, previous *routingState) {
	sample := sampleKeys(next, previous, s.keySpace)

	s.observer.ObserveTopology(models.TopologyStats{
		Version:    next.topology.Version,
		KeyShare:   sample.share,
		MovedShare: sample.moved,
		RingPoints: ringPoints(next),
	})
}

type sizedStrategy interface {
	Size() int
}

func ringPoints(state *routingState) int {
	if sized, ok := state.strategy.(sizedStrategy); ok {
		return sized.Size()
	}

	return len(state.topology.Nodes)
}

type keySample struct {
	share map[string]float64
	moved float64
}

func sampleKeys(next, previous *routingState, keySpace int64) keySample {
	step := max(keySpace/keyShareSamples, 1)
	counts := map[string]int{}
	moved := 0
	total := 0

	for id := int64(1); id <= keySpace; id += step {
		key := strconv.FormatInt(id, 10)
		owner := next.strategy.Locate(key).ID

		counts[owner]++
		total++

		if previous != nil && previous.strategy.Locate(key).ID != owner {
			moved++
		}
	}

	sample := keySample{
		share: make(map[string]float64, len(counts)),
		moved: float64(moved) / float64(total),
	}

	for node, count := range counts {
		sample.share[node] = float64(count) / float64(total)
	}

	return sample
}
