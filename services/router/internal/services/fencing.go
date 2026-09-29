package services

import (
	"context"
	"errors"
	"time"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/sharding"
)

const announceTimeout = 5 * time.Second

func (s *TopologyService) remember(version int64) {
	for {
		newest := s.newest.Load()

		if version <= newest || s.newest.CompareAndSwap(newest, version) {
			return
		}
	}
}

func (s *TopologyService) fenceVersion(current *routingState, pending *pendingChange) int64 {
	state := current

	if pending != nil {
		state = pending.next
	}

	if !fenced(state) {
		return 0
	}

	return s.newest.Load()
}

func (s *TopologyService) announce(ctx context.Context, version int64, next, previous *routingState) error {
	if !fenced(next) && !fenced(previous) {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), announceTimeout)
	defer cancel()

	var failures []error

	for _, node := range affectedNodes(next, previous) {
		if err := s.fencer.Fence(ctx, node, version, fenced(next)); err != nil {
			failures = append(failures, err)
		}
	}

	return errors.Join(failures...)
}

func fenced(state *routingState) bool {
	return state != nil && state.topology.Fencing
}

func affectedNodes(next, previous *routingState) []sharding.Node {
	seen := map[string]struct{}{}
	nodes := make([]sharding.Node, 0, len(next.topology.Nodes))

	for _, state := range []*routingState{next, previous} {
		if state == nil {
			continue
		}

		for _, node := range state.topology.Nodes {
			if _, ok := seen[node.ID]; ok {
				continue
			}

			seen[node.ID] = struct{}{}
			nodes = append(nodes, node)
		}
	}

	return nodes
}
