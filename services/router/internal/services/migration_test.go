package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/sharding"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/snapshot"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/router/internal/models"
)

type memoryCluster struct {
	mu    sync.Mutex
	nodes map[string]map[int64]string
}

func newMemoryCluster() *memoryCluster {
	return &memoryCluster{nodes: map[string]map[int64]string{}}
}

func (c *memoryCluster) put(node string, userID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.nodes[node] == nil {
		c.nodes[node] = map[int64]string{}
	}

	c.nodes[node][userID] = fmt.Sprintf(`{"user_id":%d,"wallets":[]}`, userID)
}

func (c *memoryCluster) has(node string, userID int64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, ok := c.nodes[node][userID]

	return ok
}

func (c *memoryCluster) Import(_ context.Context, node sharding.Node, _ int64, body []byte) error {
	for _, line := range splitLines(body) {
		userID, err := snapshot.UserID(line)
		if err != nil {
			return err
		}

		c.put(node.ID, userID)
	}

	return nil
}

func (c *memoryCluster) Export(_ context.Context, node sharding.Node, _ int64, visit func([]byte) error) error {
	c.mu.Lock()
	lines := make([]string, 0, len(c.nodes[node.ID]))

	for _, line := range c.nodes[node.ID] {
		lines = append(lines, line)
	}
	c.mu.Unlock()

	for _, line := range lines {
		if err := visit([]byte(line)); err != nil {
			return err
		}
	}

	return nil
}

func (c *memoryCluster) Evict(_ context.Context, node sharding.Node, _ int64, userIDs []int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, id := range userIDs {
		delete(c.nodes[node.ID], id)
	}

	return nil
}

func splitLines(body []byte) [][]byte {
	var lines [][]byte

	start := 0

	for i, b := range body {
		if b == '\n' {
			if i > start {
				lines = append(lines, body[start:i])
			}

			start = i + 1
		}
	}

	return lines
}

type migrationFixture struct {
	topology  *TopologyService
	routing   *RoutingService
	migration *MigrationService
	cluster   *memoryCluster
	observer  *recordedMigration
	before    models.Topology
	after     models.Topology
}

const fixtureUsers = 2000

func newMigrationFixture(t *testing.T, gateTimeout time.Duration) *migrationFixture {
	t.Helper()

	f := &migrationFixture{
		cluster:  newMemoryCluster(),
		observer: &recordedMigration{},
		before:   models.Topology{Version: 1, Strategy: "ring", VNodes: 16, Nodes: threeNodes()},
		after:    models.Topology{Version: 2, Strategy: "ring", VNodes: 16, Nodes: fourNodes()},
	}

	f.topology = NewTopologyService("router-1", fixtureUsers, noFencer{}, &recordedTopology{}, f.observer)
	f.routing = NewRoutingService(f.topology, f.observer, gateTimeout)
	f.migration = NewMigrationService(f.topology, f.cluster, f.observer)

	if _, err := f.topology.Apply(context.Background(), f.before); err != nil {
		t.Fatal(err)
	}

	for id := int64(1); id <= fixtureUsers; id++ {
		route, _ := f.routing.Locate(context.Background(), id)
		f.cluster.put(route.Node.ID, id)
	}

	return f
}

func (f *migrationFixture) owners(topology models.Topology) func(int64) string {
	state, err := newRoutingState(topology)
	if err != nil {
		panic(err)
	}

	return func(userID int64) string {
		return state.strategy.Locate(strconv.FormatInt(userID, 10)).ID
	}
}

func (f *migrationFixture) movingAndStaying() (int64, int64) {
	oldOwner, newOwner := f.owners(f.before), f.owners(f.after)

	var moving, staying int64

	for id := int64(1); id <= fixtureUsers && (moving == 0 || staying == 0); id++ {
		if oldOwner(id) != newOwner(id) && moving == 0 {
			moving = id
		}

		if oldOwner(id) == newOwner(id) && staying == 0 {
			staying = id
		}
	}

	return moving, staying
}

func TestMigration_FullCycle(t *testing.T) {
	ctx := context.Background()
	f := newMigrationFixture(t, 5*time.Second)
	moving, staying := f.movingAndStaying()

	f.cluster.put("wallet-4", staying)

	if _, err := f.topology.Prepare(ctx, f.after); err != nil {
		t.Fatal(err)
	}

	route, release, err := f.routing.LocateWrite(ctx, staying)
	if err != nil {
		t.Fatal(err)
	}
	release()

	if route.Version != 1 {
		t.Fatalf("user that does not move must be routed by the current topology, got version %d", route.Version)
	}

	written := make(chan models.Route)

	go func() {
		route, release, err := f.routing.LocateWrite(ctx, moving)
		if err != nil {
			t.Error(err)
		}

		release()
		written <- route
	}()

	select {
	case <-written:
		t.Fatal("write to a moving user must wait for commit")
	case <-time.After(100 * time.Millisecond):
	}

	result, err := f.migration.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	newOwner, oldOwner := f.owners(f.after), f.owners(f.before)

	if !f.cluster.has(newOwner(moving), moving) {
		t.Fatalf("user %d was not copied to %s", moving, newOwner(moving))
	}

	if result.Users == 0 || result.PerNode["wallet-4"] != result.Users {
		t.Fatalf("result = %+v, every moved user must go to the new node", result)
	}

	if _, err := f.topology.Commit(ctx, 2); err != nil {
		t.Fatal(err)
	}

	select {
	case route := <-written:
		if route.Version != 2 || route.Node.ID != newOwner(moving) {
			t.Fatalf("waiting write must go to the new owner after commit, got %+v", route)
		}
	case <-time.After(time.Second):
		t.Fatal("waiting write was not released by commit")
	}

	cleaned, err := f.migration.Cleanup(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if cleaned.Users != result.Users || f.cluster.has(oldOwner(moving), moving) {
		t.Fatalf("cleanup removed %d of %d users, user %d still on %s", cleaned.Users, result.Users, moving, oldOwner(moving))
	}

	if !f.cluster.has("wallet-4", staying) {
		t.Fatal("stale copy on a node that never owned the user must not be touched")
	}
}

func TestMigration_GateTimeout(t *testing.T) {
	ctx := context.Background()
	f := newMigrationFixture(t, 50*time.Millisecond)
	moving, _ := f.movingAndStaying()

	if _, err := f.topology.Prepare(ctx, f.after); err != nil {
		t.Fatal(err)
	}

	_, _, err := f.routing.LocateWrite(ctx, moving)
	if !errors.Is(err, models.ErrMigrationTimeout) {
		t.Fatalf("err = %v, want ErrMigrationTimeout", err)
	}

	if f.observer.rejected != 1 || f.observer.waiting != 0 {
		t.Fatalf("rejected = %d, waiting = %d", f.observer.rejected, f.observer.waiting)
	}
}

func TestMigration_PhaseRules(t *testing.T) {
	ctx := context.Background()
	f := newMigrationFixture(t, time.Second)

	if _, err := f.topology.Commit(ctx, 2); !errors.Is(err, models.ErrNoMigration) {
		t.Fatalf("commit without prepare: err = %v", err)
	}

	if _, err := f.migration.Run(ctx); !errors.Is(err, models.ErrNoMigration) {
		t.Fatalf("run without prepare: err = %v", err)
	}

	if _, err := f.topology.Prepare(ctx, f.after); err != nil {
		t.Fatal(err)
	}

	if !f.observer.active {
		t.Fatal("prepare must mark migration active")
	}

	if _, err := f.topology.Apply(ctx, models.Topology{Version: 3, Strategy: "mod", Nodes: threeNodes()}); !errors.Is(err, models.ErrMigrationInProgress) {
		t.Fatalf("apply during migration: err = %v", err)
	}

	if _, err := f.migration.Cleanup(ctx); !errors.Is(err, models.ErrMigrationInProgress) {
		t.Fatalf("cleanup before commit: err = %v", err)
	}

	if _, err := f.topology.Commit(ctx, 3); !errors.Is(err, models.ErrVersionMismatch) {
		t.Fatalf("commit of another version: err = %v", err)
	}

	moving, _ := f.movingAndStaying()

	if _, err := f.migration.Run(ctx); err != nil {
		t.Fatal(err)
	}

	if _, err := f.topology.Abort(ctx); err != nil {
		t.Fatal(err)
	}

	current, _ := f.topology.Current(ctx)
	if current.Version != 1 || f.observer.active {
		t.Fatalf("abort must keep the old topology, got version %d", current.Version)
	}

	cleaned, err := f.migration.Cleanup(ctx)
	if err != nil || cleaned.Users != 0 {
		t.Fatalf("cleanup after abort must not touch anyone: %+v, err %v", cleaned, err)
	}

	if !f.cluster.has(f.owners(f.before)(moving), moving) {
		t.Fatal("aborted migration must leave users on their current owners")
	}
}
