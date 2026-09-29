package services

import (
	"context"
	"errors"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/sharding"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/router/internal/models"
)

type recordedTopology struct {
	version    int64
	keyShare   map[string]float64
	movedShare float64
	ringPoints int
}

func (r *recordedTopology) ObserveTopology(stats models.TopologyStats) {
	r.version = stats.Version
	r.keyShare = stats.KeyShare
	r.movedShare = stats.MovedShare
	r.ringPoints = stats.RingPoints
}

type recordedMigration struct {
	mu       sync.Mutex
	active   bool
	copied   int
	waiting  int
	rejected int
}

func (r *recordedMigration) ObserveMigrationActive(active bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.active = active
}

func (r *recordedMigration) ObserveCopied(users int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.copied += users
}

func (r *recordedMigration) ObserveGateWaiting(delta int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.waiting += delta
}

func (r *recordedMigration) ObserveGateRejected() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.rejected++
}

type noFencer struct{}

func (noFencer) Fence(context.Context, sharding.Node, int64, bool) error {
	return nil
}

func threeNodes() []sharding.Node {
	return []sharding.Node{
		{ID: "wallet-1", Addr: "http://wallet-1:8080"},
		{ID: "wallet-2", Addr: "http://wallet-2:8080"},
		{ID: "wallet-3", Addr: "http://wallet-3:8080"},
	}
}

func TestTopologyService_Apply(t *testing.T) {
	observer := &recordedTopology{}
	svc := NewTopologyService("router-1", 30_000, noFencer{}, observer, &recordedMigration{})

	_, err := svc.Apply(context.Background(), models.Topology{Version: 1, Strategy: "mod", Nodes: threeNodes()})
	if err != nil {
		t.Fatal(err)
	}

	current, err := svc.Current(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if current.Version != 1 || current.Nodes[0].Weight != 1 {
		t.Fatalf("current = %+v", current)
	}

	if observer.version != 1 || len(observer.keyShare) != 3 {
		t.Fatalf("observed = %+v", observer)
	}

	total := 0.0

	for _, share := range observer.keyShare {
		total += share
	}

	if math.Abs(total-1) > 1e-9 {
		t.Fatalf("key share sums to %f", total)
	}
}

func TestTopologyService_ApplyRejects(t *testing.T) {
	tests := []struct {
		name     string
		topology models.Topology
		wantErr  error
	}{
		{
			name:     "same version",
			topology: models.Topology{Version: 5, Strategy: "mod", Nodes: threeNodes()},
			wantErr:  models.ErrStaleTopology,
		},
		{
			name:     "older version",
			topology: models.Topology{Version: 4, Strategy: "mod", Nodes: threeNodes()},
			wantErr:  models.ErrStaleTopology,
		},
		{
			name:     "zero version",
			topology: models.Topology{Strategy: "mod", Nodes: threeNodes()},
			wantErr:  models.ErrInvalidArgument,
		},
		{
			name:     "unknown strategy",
			topology: models.Topology{Version: 6, Strategy: "magic", Nodes: threeNodes()},
			wantErr:  models.ErrInvalidArgument,
		},
		{
			name:     "no nodes",
			topology: models.Topology{Version: 6, Strategy: "mod"},
			wantErr:  models.ErrInvalidArgument,
		},
		{
			name: "duplicate node",
			topology: models.Topology{Version: 6, Strategy: "mod", Nodes: []sharding.Node{
				{ID: "wallet-1", Addr: "http://wallet-1:8080"},
				{ID: "wallet-1", Addr: "http://wallet-1:8080"},
			}},
			wantErr: models.ErrInvalidArgument,
		},
		{
			name: "addr without scheme",
			topology: models.Topology{Version: 6, Strategy: "mod", Nodes: []sharding.Node{
				{ID: "wallet-1", Addr: "wallet-1:8080"},
			}},
			wantErr: models.ErrInvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewTopologyService("router-1", 1000, noFencer{}, &recordedTopology{}, &recordedMigration{})

			_, err := svc.Apply(context.Background(), models.Topology{Version: 5, Strategy: "mod", Nodes: threeNodes()})
			if err != nil {
				t.Fatal(err)
			}

			if _, err := svc.Apply(context.Background(), tt.topology); !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}

			current, _ := svc.Current(context.Background())
			if current.Version != 5 {
				t.Fatalf("rejected topology changed the current one: %+v", current)
			}
		})
	}
}

func TestRoutingService_Locate(t *testing.T) {
	topology := NewTopologyService("router-1", 1000, noFencer{}, &recordedTopology{}, &recordedMigration{})
	routing := NewRoutingService(topology, &recordedMigration{}, time.Second)

	if _, err := routing.Locate(context.Background(), 42); !errors.Is(err, models.ErrNoTopology) {
		t.Fatalf("err = %v, want ErrNoTopology", err)
	}

	_, err := topology.Apply(context.Background(), models.Topology{Version: 3, Strategy: "mod", Nodes: threeNodes()})
	if err != nil {
		t.Fatal(err)
	}

	route, err := routing.Locate(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}

	want := threeNodes()[sharding.Hash("42")%3].ID

	if route.Key != "42" || route.Node.ID != want || route.Version != 3 || route.Strategy != "mod" {
		t.Fatalf("route = %+v, want node %s", route, want)
	}
}

func fourNodes() []sharding.Node {
	return append(threeNodes(), sharding.Node{ID: "wallet-4", Addr: "http://wallet-4:8080"})
}

func TestTopologyService_MovedShare(t *testing.T) {
	tests := []struct {
		strategy string
		want     func(observer *recordedTopology) float64
	}{
		{"mod", func(*recordedTopology) float64 { return 0.75 }},
		{"ring", func(observer *recordedTopology) float64 { return observer.keyShare["wallet-4"] }},
	}

	for _, tt := range tests {
		t.Run(tt.strategy, func(t *testing.T) {
			observer := &recordedTopology{}
			svc := NewTopologyService("router-1", 200_000, noFencer{}, observer, &recordedMigration{})
			ctx := context.Background()

			if _, err := svc.Apply(ctx, models.Topology{Version: 1, Strategy: tt.strategy, Nodes: threeNodes()}); err != nil {
				t.Fatal(err)
			}

			if observer.movedShare != 0 {
				t.Fatalf("first topology moved %.3f, want 0", observer.movedShare)
			}

			if _, err := svc.Apply(ctx, models.Topology{Version: 2, Strategy: tt.strategy, Nodes: fourNodes()}); err != nil {
				t.Fatal(err)
			}

			if want := tt.want(observer); math.Abs(observer.movedShare-want) > 0.01 {
				t.Fatalf("moved %.3f, want about %.3f", observer.movedShare, want)
			}
		})
	}
}

func TestRoutingService_RingPoint(t *testing.T) {
	ctx := context.Background()
	topology := NewTopologyService("router-1", 1000, noFencer{}, &recordedTopology{}, &recordedMigration{})
	routing := NewRoutingService(topology, &recordedMigration{}, time.Second)

	if _, err := topology.Apply(ctx, models.Topology{Version: 1, Strategy: "mod", Nodes: threeNodes()}); err != nil {
		t.Fatal(err)
	}

	if route, _ := routing.Locate(ctx, 42); route.Point != nil {
		t.Fatalf("mod must not have a ring point: %+v", route.Point)
	}

	if _, err := topology.Apply(ctx, models.Topology{Version: 2, Strategy: "ring", VNodes: 4, Nodes: threeNodes()}); err != nil {
		t.Fatal(err)
	}

	route, err := routing.Locate(ctx, 42)
	if err != nil {
		t.Fatal(err)
	}

	if route.Point == nil || route.Point.Node != route.Node.ID || route.Point.Index < 0 || route.Point.Index >= 4 {
		t.Fatalf("route = %+v, point = %+v", route, route.Point)
	}

	current, _ := topology.Current(ctx)
	if current.VNodes != 4 {
		t.Fatalf("vnodes = %d, want 4", current.VNodes)
	}
}

func TestTopologyService_RejectsVNodesOutOfRange(t *testing.T) {
	svc := NewTopologyService("router-1", 1000, noFencer{}, &recordedTopology{}, &recordedMigration{})

	for _, vnodes := range []int{-1, maxVNodes + 1} {
		_, err := svc.Apply(context.Background(), models.Topology{Version: 1, Strategy: "ring", VNodes: vnodes, Nodes: threeNodes()})
		if !errors.Is(err, models.ErrInvalidArgument) {
			t.Fatalf("vnodes %d: err = %v, want ErrInvalidArgument", vnodes, err)
		}
	}
}

func TestTopologyService_RingPoints(t *testing.T) {
	ctx := context.Background()
	observer := &recordedTopology{}
	svc := NewTopologyService("router-1", 1000, noFencer{}, observer, &recordedMigration{})

	weighted := fourNodes()
	weighted[3].Weight = 2

	steps := []struct {
		topology models.Topology
		want     int
	}{
		{models.Topology{Version: 1, Strategy: "mod", Nodes: fourNodes()}, 4},
		{models.Topology{Version: 2, Strategy: "ring", VNodes: 16, Nodes: fourNodes()}, 64},
		{models.Topology{Version: 3, Strategy: "ring", VNodes: 256, Nodes: weighted}, 1280},
	}

	for _, step := range steps {
		if _, err := svc.Apply(ctx, step.topology); err != nil {
			t.Fatal(err)
		}

		if observer.ringPoints != step.want {
			t.Fatalf("version %d: ring points = %d, want %d", step.topology.Version, observer.ringPoints, step.want)
		}
	}
}
