package client

import (
	"context"
	"net/http"
	"time"
)

const (
	longTimeout = 10 * time.Minute

	StrategyRing = "ring"
)

type Node struct {
	ID     string `json:"id"`
	Addr   string `json:"addr"`
	Weight int    `json:"weight,omitempty"`
}

type Topology struct {
	Version         int64             `json:"version"`
	PendingVersion  int64             `json:"pending_version,omitempty"`
	DeferredError   string            `json:"deferred_error,omitempty"`
	Strategy        string            `json:"strategy"`
	VNodes          int               `json:"vnodes,omitempty"`
	Fencing         bool              `json:"fencing,omitempty"`
	MembershipDelay map[string]string `json:"membership_delay,omitempty"`
	Nodes           []Node            `json:"nodes"`
}

type Change struct {
	DeferredFor string `json:"deferred_for"`
	Topology
}

func (c Change) Deferred() bool {
	return c.DeferredFor != ""
}

type Spread struct {
	Users   int            `json:"users"`
	PerNode map[string]int `json:"per_node"`
}

type Router struct {
	endpoint
}

func NewRouter(url string, client *http.Client) *Router {
	return &Router{endpoint: newEndpoint("router", url, client)}
}

func (r *Router) Topology(ctx context.Context) (Topology, bool, error) {
	var topology Topology

	err := r.get(ctx, "/topology", &topology)
	if HasStatus(err, http.StatusServiceUnavailable) {
		return Topology{}, false, nil
	}

	if err != nil {
		return Topology{}, false, err
	}

	return topology, true, nil
}

func (r *Router) Apply(ctx context.Context, topology Topology) (Change, error) {
	return r.change(ctx, "/topology", topology)
}

func (r *Router) Prepare(ctx context.Context, topology Topology) (Change, error) {
	return r.change(ctx, "/topology/prepare", topology)
}

func (r *Router) Commit(ctx context.Context, version int64) (Change, error) {
	return r.change(ctx, "/topology/commit", map[string]int64{"version": version})
}

func (r *Router) Abort(ctx context.Context) (Change, error) {
	return r.change(ctx, "/topology/abort", nil)
}

func (r *Router) Migrate(ctx context.Context) (Spread, error) {
	var copied Spread

	err := r.send(ctx, http.MethodPost, "/migrate", nil, longTimeout, &copied)

	return copied, err
}

func (r *Router) Cleanup(ctx context.Context) (int, error) {
	var cleaned struct {
		Users int `json:"users"`
	}

	err := r.send(ctx, http.MethodPost, "/migrate/cleanup", nil, longTimeout, &cleaned)

	return cleaned.Users, err
}

func (r *Router) Bootstrap(ctx context.Context) (Spread, error) {
	var spread Spread

	err := r.send(ctx, http.MethodPost, "/bootstrap", nil, longTimeout, &spread)

	return spread, err
}

func (r *Router) change(ctx context.Context, path string, payload any) (Change, error) {
	var change Change

	err := r.send(ctx, http.MethodPost, path, payload, quickTimeout, &change)

	return change, err
}
