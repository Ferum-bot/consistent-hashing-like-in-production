package api

import (
	"net/http"
	"time"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/request"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/sharding"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/router/internal/models"
)

type topologyDTO struct {
	Version         int64             `json:"version"`
	PendingVersion  int64             `json:"pending_version,omitempty"`
	DeferredError   string            `json:"deferred_error,omitempty"`
	Strategy        string            `json:"strategy"`
	VNodes          int               `json:"vnodes,omitempty"`
	Fencing         bool              `json:"fencing,omitempty"`
	MembershipDelay map[string]string `json:"membership_delay,omitempty"`
	Nodes           []nodeDTO         `json:"nodes"`
}

type deferredResponse struct {
	DeferredFor string `json:"deferred_for"`
}

type commitRequest struct {
	Version int64 `json:"version"`
}

type migrationResponse struct {
	Users   int            `json:"users"`
	PerNode map[string]int `json:"per_node"`
}

type cleanupResponse struct {
	Users int `json:"users"`
}

type nodeDTO struct {
	ID     string `json:"id"`
	Addr   string `json:"addr"`
	Weight int    `json:"weight"`
}

type routeDTO struct {
	UserID   int64         `json:"user_id"`
	Key      string        `json:"key"`
	Hash     uint64        `json:"hash"`
	Node     string        `json:"node"`
	Addr     string        `json:"addr"`
	Strategy string        `json:"strategy"`
	Version  int64         `json:"topology_version"`
	Point    *ringPointDTO `json:"point,omitempty"`
}

type ringPointDTO struct {
	Name string `json:"name"`
	Node string `json:"node"`
	Hash uint64 `json:"hash"`
}

type bootstrapResponse struct {
	Users   int            `json:"users"`
	PerNode map[string]int `json:"per_node"`
}

func toTopologyDTO(topology models.Topology) topologyDTO {
	nodes := make([]nodeDTO, 0, len(topology.Nodes))

	for _, node := range topology.Nodes {
		nodes = append(nodes, nodeDTO{ID: node.ID, Addr: node.Addr, Weight: node.Weight})
	}

	dto := topologyDTO{
		Version:  topology.Version,
		Strategy: topology.Strategy,
		VNodes:   vnodesOf(topology),
		Fencing:  topology.Fencing,
		Nodes:    nodes,
	}

	for router, delay := range topology.MembershipDelay {
		if dto.MembershipDelay == nil {
			dto.MembershipDelay = map[string]string{}
		}

		dto.MembershipDelay[router] = delay.String()
	}

	return dto
}

func toStateDTO(state models.RouterState) topologyDTO {
	dto := toTopologyDTO(state.Topology)
	dto.PendingVersion = state.PendingVersion
	dto.DeferredError = state.DeferredError

	return dto
}

func (d topologyDTO) toModel() (models.Topology, error) {
	nodes := make([]sharding.Node, 0, len(d.Nodes))

	for _, node := range d.Nodes {
		nodes = append(nodes, sharding.Node{ID: node.ID, Addr: node.Addr, Weight: node.Weight})
	}

	delays := make(map[string]time.Duration, len(d.MembershipDelay))

	for router, raw := range d.MembershipDelay {
		delay, err := time.ParseDuration(raw)
		if err != nil {
			return models.Topology{}, models.NewErrInvalidDelay(router, raw)
		}

		delays[router] = delay
	}

	return models.Topology{
		Version:         d.Version,
		Strategy:        d.Strategy,
		VNodes:          d.VNodes,
		Nodes:           nodes,
		MembershipDelay: delays,
		Fencing:         d.Fencing,
	}, nil
}

func decodeTopology(r *http.Request) (models.Topology, error) {
	var dto topologyDTO
	if err := request.DecodeJSON(r, &dto); err != nil {
		return models.Topology{}, err
	}

	return dto.toModel()
}

func toRouteDTO(route models.Route) routeDTO {
	dto := routeDTO{
		UserID:   route.UserID,
		Key:      route.Key,
		Hash:     route.Hash,
		Node:     route.Node.ID,
		Addr:     route.Node.Addr,
		Strategy: route.Strategy,
		Version:  route.Version,
	}

	if route.Point != nil {
		dto.Point = &ringPointDTO{
			Name: sharding.PointKey(route.Point.Node, route.Point.Index),
			Node: route.Point.Node,
			Hash: route.Point.Hash,
		}
	}

	return dto
}

func vnodesOf(topology models.Topology) int {
	if topology.Strategy != models.StrategyRing {
		return 0
	}

	return topology.VNodes
}

func toBootstrapResponse(result models.BootstrapResult) bootstrapResponse {
	return bootstrapResponse{Users: result.Users, PerNode: result.PerNode}
}
