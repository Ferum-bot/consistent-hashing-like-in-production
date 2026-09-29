package services

import (
	"net/url"
	"time"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/sharding"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/router/internal/models"
)

const (
	maxVNodes          = 10_000
	maxRingPoints      = 1_000_000
	maxMembershipDelay = 10 * time.Minute
)

func newRoutingState(topology models.Topology) (*routingState, error) {
	topology = withDefaults(topology)

	if err := validateTopology(topology); err != nil {
		return nil, err
	}

	strategy, err := buildStrategy(topology)
	if err != nil {
		return nil, err
	}

	return &routingState{topology: topology, strategy: strategy}, nil
}

func withDefaults(topology models.Topology) models.Topology {
	if topology.VNodes == 0 {
		topology.VNodes = 1
	}

	nodes := make([]sharding.Node, len(topology.Nodes))

	for i, node := range topology.Nodes {
		if node.Weight == 0 {
			node.Weight = 1
		}

		nodes[i] = node
	}

	topology.Nodes = nodes

	return topology
}

func validateTopology(topology models.Topology) error {
	if topology.Version <= 0 {
		return models.NewErrVersionMustBePositive(topology.Version)
	}

	if len(topology.Nodes) == 0 {
		return models.NewErrNoNodes()
	}

	if topology.VNodes < 1 || topology.VNodes > maxVNodes {
		return models.NewErrVNodesOutOfRange(topology.VNodes, maxVNodes)
	}

	for router, delay := range topology.MembershipDelay {
		if delay < 0 || delay > maxMembershipDelay {
			return models.NewErrDelayOutOfRange(router, delay, maxMembershipDelay)
		}
	}

	seen := make(map[string]struct{}, len(topology.Nodes))

	for _, node := range topology.Nodes {
		if _, ok := seen[node.ID]; ok {
			return models.NewErrDuplicateNode(node.ID)
		}

		seen[node.ID] = struct{}{}

		if err := validateNode(node); err != nil {
			return err
		}
	}

	if points := ringPointsOf(topology); points > maxRingPoints {
		return models.NewErrTooManyRingPoints(points, maxRingPoints)
	}

	return nil
}

func validateNode(node sharding.Node) error {
	if node.ID == "" {
		return models.NewErrInvalidNode(node.ID, "empty id")
	}

	addr, err := url.Parse(node.Addr)
	if err != nil || addr.Scheme == "" || addr.Host == "" {
		return models.NewErrInvalidNode(node.ID, "addr must look like http://host:port")
	}

	if node.Weight < 0 || node.Weight > maxRingPoints {
		return models.NewErrInvalidNode(node.ID, "weight out of range")
	}

	return nil
}

func ringPointsOf(topology models.Topology) int {
	points := 0

	for _, node := range topology.Nodes {
		points += topology.VNodes * node.Weight
	}

	return points
}
