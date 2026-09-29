package services

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/sharding"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/router/internal/models"
)

type fenceCall struct {
	node    string
	version int64
	strict  bool
}

type recordingFencer struct {
	mu    sync.Mutex
	calls []fenceCall
}

func (f *recordingFencer) Fence(_ context.Context, node sharding.Node, version int64, strict bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, fenceCall{node: node.ID, version: version, strict: strict})

	return nil
}

func (f *recordingFencer) nodes() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	var nodes []string

	for _, call := range f.calls {
		nodes = append(nodes, call.node)
	}

	sort.Strings(nodes)

	return nodes
}

func TestMembershipDelay_RouterAppliesLate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lagging := NewTopologyService("router-2", 1000, noFencer{}, &recordedTopology{}, &recordedMigration{})
	go lagging.RunDeferred(ctx)

	if _, err := lagging.Apply(ctx, models.Topology{Version: 1, Strategy: "ring", Nodes: threeNodes()}); err != nil {
		t.Fatal(err)
	}

	delay := map[string]time.Duration{"router-2": 150 * time.Millisecond}

	deferred, err := lagging.Prepare(ctx, models.Topology{Version: 2, Strategy: "ring", Nodes: fourNodes(), MembershipDelay: delay})
	if err != nil {
		t.Fatal(err)
	}

	if deferred != 150*time.Millisecond {
		t.Fatalf("prepare deferred for %s, want 150ms", deferred)
	}

	time.Sleep(50 * time.Millisecond)

	deferred, err = lagging.Commit(ctx, 2)
	if err != nil || deferred == 0 || deferred > 110*time.Millisecond {
		t.Fatalf("commit must wait only until the router catches up, deferred %s, err %v", deferred, err)
	}

	current, _ := lagging.Current(ctx)
	if current.Version != 1 {
		t.Fatalf("lagging router switched too early: version %d", current.Version)
	}

	deadline := time.Now().Add(2 * time.Second)

	for time.Now().Before(deadline) {
		if current, _ := lagging.Current(ctx); current.Version == 2 {
			break
		}

		time.Sleep(10 * time.Millisecond)
	}

	if current, _ := lagging.Current(ctx); current.Version != 2 {
		t.Fatal("lagging router never applied prepare and commit")
	}

	deferred, err = lagging.Commit(ctx, 3)
	if deferred != 0 || !errors.Is(err, models.ErrNoMigration) {
		t.Fatalf("after catching up the router must answer at once, deferred %s, err %v", deferred, err)
	}
}

func TestMembershipDelay_OtherRoutersApplyAtOnce(t *testing.T) {
	ctx := context.Background()
	svc := NewTopologyService("router-1", 1000, noFencer{}, &recordedTopology{}, &recordedMigration{})

	delay := map[string]time.Duration{"router-2": time.Minute}

	deferred, err := svc.Apply(ctx, models.Topology{Version: 1, Strategy: "ring", Nodes: threeNodes(), MembershipDelay: delay})
	if err != nil || deferred != 0 {
		t.Fatalf("router-1 must apply at once: deferred %s, err %v", deferred, err)
	}
}

func TestFencing_AnnouncesVersionToEveryNode(t *testing.T) {
	ctx := context.Background()
	fencer := &recordingFencer{}
	topology := NewTopologyService("router-1", 1000, fencer, &recordedTopology{}, &recordedMigration{})
	routing := NewRoutingService(topology, &recordedMigration{}, time.Second)

	if _, err := topology.Apply(ctx, models.Topology{Version: 1, Strategy: "ring", Nodes: threeNodes()}); err != nil {
		t.Fatal(err)
	}

	if route, _ := routing.Locate(ctx, 42); route.FenceVersion != 0 {
		t.Fatalf("without fencing the router must not send a version, got %d", route.FenceVersion)
	}

	fencer.calls = nil

	if _, err := topology.Prepare(ctx, models.Topology{Version: 2, Strategy: "ring", Nodes: fourNodes(), Fencing: true}); err != nil {
		t.Fatal(err)
	}

	if got := fencer.nodes(); len(got) != 4 || got[3] != "wallet-4" {
		t.Fatalf("version must be announced to old and new nodes, got %v", got)
	}

	for _, call := range fencer.calls {
		if call.version != 2 || !call.strict {
			t.Fatalf("call = %+v, want version 2 in strict mode", call)
		}
	}

	route, err := routing.Locate(ctx, 42)
	if err != nil {
		t.Fatal(err)
	}

	if route.Version != 1 || route.FenceVersion != 2 {
		t.Fatalf("during migration route by version 1 but send version 2, got %+v", route)
	}
}

func TestFencing_AbortKeepsNodesReachable(t *testing.T) {
	ctx := context.Background()
	fencer := &recordingFencer{}
	topology := NewTopologyService("router-1", 1000, fencer, &recordedTopology{}, &recordedMigration{})
	routing := NewRoutingService(topology, &recordedMigration{}, time.Second)

	if _, err := topology.Apply(ctx, models.Topology{Version: 1, Strategy: "ring", Nodes: threeNodes(), Fencing: true}); err != nil {
		t.Fatal(err)
	}

	if _, err := topology.Prepare(ctx, models.Topology{Version: 2, Strategy: "ring", Nodes: fourNodes(), Fencing: true}); err != nil {
		t.Fatal(err)
	}

	if _, err := topology.Abort(ctx); err != nil {
		t.Fatal(err)
	}

	route, _ := routing.Locate(ctx, 42)
	if route.Version != 1 || route.FenceVersion != 2 {
		t.Fatalf("nodes already saw version 2, the router must keep sending it after abort, got %+v", route)
	}

	fencer.calls = nil

	if _, err := topology.Apply(ctx, models.Topology{Version: 3, Strategy: "ring", Nodes: threeNodes()}); err != nil {
		t.Fatal(err)
	}

	for _, call := range fencer.calls {
		if call.version != 3 || call.strict {
			t.Fatalf("turning fencing off must release strict mode on nodes, got %+v", call)
		}
	}

	if route, _ := routing.Locate(ctx, 42); route.FenceVersion != 0 {
		t.Fatalf("without fencing the router must not send a version, got %d", route.FenceVersion)
	}
}

type failingFencer struct{}

func (failingFencer) Fence(context.Context, sharding.Node, int64, bool) error {
	return errors.New("wallet-3 is down")
}

func TestMembershipDelay_FailedChangeIsVisible(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lagging := NewTopologyService("router-2", 1000, failingFencer{}, &recordedTopology{}, &recordedMigration{})
	go lagging.RunDeferred(ctx)

	if _, err := lagging.Apply(ctx, models.Topology{Version: 1, Strategy: "ring", Nodes: threeNodes()}); err != nil {
		t.Fatal(err)
	}

	delay := map[string]time.Duration{"router-2": 20 * time.Millisecond}

	if _, err := lagging.Prepare(ctx, models.Topology{Version: 2, Strategy: "ring", Nodes: fourNodes(), Fencing: true, MembershipDelay: delay}); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)

	for time.Now().Before(deadline) {
		state, _ := lagging.State(ctx)
		if state.DeferredError != "" {
			if state.Topology.Version != 1 || state.PendingVersion != 0 {
				t.Fatalf("failed prepare must leave the router on version 1, got %+v", state)
			}

			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("a failed deferred change must be reported in the router state")
}

func TestFencing_FailedApplyKeepsTheOldVersion(t *testing.T) {
	ctx := context.Background()
	topology := NewTopologyService("router-1", 1000, failingFencer{}, &recordedTopology{}, &recordedMigration{})
	routing := NewRoutingService(topology, &recordedMigration{}, time.Second)

	if _, err := topology.Apply(ctx, models.Topology{Version: 1, Strategy: "ring", Nodes: threeNodes()}); err != nil {
		t.Fatal(err)
	}

	if _, err := topology.Apply(ctx, models.Topology{Version: 2, Strategy: "ring", Nodes: fourNodes(), Fencing: true}); err == nil {
		t.Fatal("apply must fail when nodes do not accept the version")
	}

	route, _ := routing.Locate(ctx, 42)
	if route.Version != 1 || route.FenceVersion != 0 {
		t.Fatalf("failed apply must not change routing or the fence version, got %+v", route)
	}
}
