package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/tools/chctl/client"
)

type topologyFile struct {
	client.Topology
	Migrate bool `json:"migrate"`
}

type routerState struct {
	router   *client.Router
	topology client.Topology
	set      bool
}

func readTopologyFile(path string) (topologyFile, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return topologyFile{}, withHint(err, "scenarios keep it in scenarios/<name>/topology.json")
	}

	if err != nil {
		return topologyFile{}, err
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var file topologyFile
	if err := decoder.Decode(&file); err != nil {
		return topologyFile{}, fmt.Errorf("%s: %w", path, err)
	}

	if file.Version != 0 || file.PendingVersion != 0 {
		return topologyFile{}, fmt.Errorf("%s: remove the version, chctl assigns it", path)
	}

	if len(file.Nodes) == 0 {
		return topologyFile{}, fmt.Errorf("%s: no nodes", path)
	}

	return file, nil
}

func readRouters(ctx context.Context, routers []*client.Router) ([]routerState, error) {
	states := make([]routerState, 0, len(routers))

	for _, router := range routers {
		topology, set, err := router.Topology(ctx)
		if err != nil {
			return nil, err
		}

		states = append(states, routerState{router: router, topology: topology, set: set})
	}

	return states, nil
}

func nextVersion(states []routerState) int64 {
	var latest int64

	for _, state := range states {
		latest = max(latest, state.topology.PendingVersion)
	}

	return max(latest, newestVersion(states)) + 1
}

func newestState(states []routerState) (routerState, bool) {
	var newest routerState

	for _, state := range states {
		if state.set && (!newest.set || state.topology.Version > newest.topology.Version) {
			newest = state
		}
	}

	return newest, newest.set
}

func newestVersion(states []routerState) int64 {
	var newest int64

	for _, state := range states {
		newest = max(newest, state.topology.Version)
	}

	return newest
}

func unfinishedMigration(states []routerState) error {
	for _, state := range states {
		if state.topology.PendingVersion != 0 {
			err := fmt.Errorf("%s has an unfinished migration to v%d", state.router.URL(), state.topology.PendingVersion)

			return withHint(err, hintAbort)
		}
	}

	return nil
}

func anyTopology(states []routerState) bool {
	for _, state := range states {
		if state.set {
			return true
		}
	}

	return false
}

func describeTopology(topology client.Topology) string {
	parts := []string{topology.Strategy}

	if topology.Strategy == client.StrategyRing {
		parts = append(parts, fmt.Sprintf("%d vnodes", topology.VNodes))
	}

	parts = append(parts, fmt.Sprintf("%d nodes", len(topology.Nodes)))

	if topology.Fencing {
		parts = append(parts, "fencing on")
	}

	routers := make([]string, 0, len(topology.MembershipDelay))

	for router := range topology.MembershipDelay {
		routers = append(routers, router)
	}

	sort.Strings(routers)

	for _, router := range routers {
		parts = append(parts, fmt.Sprintf("%s lags %s", router, topology.MembershipDelay[router]))
	}

	return strings.Join(parts, ", ")
}

func describeChange(change client.Change) string {
	if change.Deferred() {
		return "accepted, the router lags and applies it in " + change.DeferredFor
	}

	if change.PendingVersion != 0 {
		return fmt.Sprintf("v%d prepared, still routing by v%d", change.PendingVersion, change.Version)
	}

	return fmt.Sprintf("v%d: %s", change.Version, describeTopology(change.Topology))
}

func describeSpread(spread client.Spread) string {
	nodes := make([]string, 0, len(spread.PerNode))

	for node := range spread.PerNode {
		nodes = append(nodes, node)
	}

	sort.Strings(nodes)

	if len(nodes) == 0 {
		return ""
	}

	parts := make([]string, 0, len(nodes))

	for _, node := range nodes {
		parts = append(parts, fmt.Sprintf("%s %d", node, spread.PerNode[node]))
	}

	return ": " + strings.Join(parts, ", ")
}
