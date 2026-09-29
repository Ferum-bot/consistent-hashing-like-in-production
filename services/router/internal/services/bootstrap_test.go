package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/sharding"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/snapshot"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/router/internal/models"
)

type recordingWallets struct {
	mu      sync.Mutex
	imports map[string][]string
}

func (w *recordingWallets) Export(context.Context, sharding.Node, int64, func([]byte) error) error {
	return nil
}

func (w *recordingWallets) Evict(context.Context, sharding.Node, int64, []int64) error {
	return nil
}

func (w *recordingWallets) Import(_ context.Context, node sharding.Node, _ int64, body []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	w.imports[node.ID] = append(w.imports[node.ID], lines...)

	return nil
}

func writeSnapshot(t *testing.T, users int) string {
	t.Helper()

	var body bytes.Buffer

	for id := 1; id <= users; id++ {
		fmt.Fprintf(&body, `{"user_id":%d,"wallets":[{"currency":"USD","balance":%d}]}`+"\n", id, id*10)
	}

	path := filepath.Join(t.TempDir(), "snapshot.ndjson")

	if err := os.WriteFile(path, body.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestBootstrapService_SendsEveryUserToItsOwner(t *testing.T) {
	ctx := context.Background()
	users := usersPerRequest*2 + 123

	topology := NewTopologyService("router-1", int64(users), noFencer{}, &recordedTopology{}, &recordedMigration{})
	if _, err := topology.Apply(ctx, models.Topology{Version: 1, Strategy: "mod", Nodes: threeNodes()}); err != nil {
		t.Fatal(err)
	}

	wallets := &recordingWallets{imports: map[string][]string{}}
	bootstrap := NewBootstrapService(topology, wallets, writeSnapshot(t, users))

	result, err := bootstrap.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if result.Users != users {
		t.Fatalf("users = %d, want %d", result.Users, users)
	}

	routing := NewRoutingService(topology, &recordedMigration{}, time.Second)
	sent := 0

	for node, lines := range wallets.imports {
		if len(lines) != result.PerNode[node] {
			t.Fatalf("%s got %d lines, result says %d", node, len(lines), result.PerNode[node])
		}

		for _, line := range lines {
			userID, err := snapshot.UserID([]byte(line))
			if err != nil {
				t.Fatal(err)
			}

			route, _ := routing.Locate(ctx, userID)
			if route.Node.ID != node {
				t.Fatalf("user %d sent to %s, owner is %s", userID, node, route.Node.ID)
			}
		}

		sent += len(lines)
	}

	if sent != users {
		t.Fatalf("sent %d users, want %d", sent, users)
	}
}

func TestBootstrapService_Errors(t *testing.T) {
	ctx := context.Background()
	wallets := &recordingWallets{imports: map[string][]string{}}

	noTopology := NewBootstrapService(NewTopologyService("router-1", 10, noFencer{}, &recordedTopology{}, &recordedMigration{}), wallets, writeSnapshot(t, 1))

	if _, err := noTopology.Run(ctx); !errors.Is(err, models.ErrNoTopology) {
		t.Fatalf("err = %v, want ErrNoTopology", err)
	}

	topology := NewTopologyService("router-1", 10, noFencer{}, &recordedTopology{}, &recordedMigration{})
	if _, err := topology.Apply(ctx, models.Topology{Version: 1, Strategy: "mod", Nodes: threeNodes()}); err != nil {
		t.Fatal(err)
	}

	missing := NewBootstrapService(topology, wallets, filepath.Join(t.TempDir(), "missing.ndjson"))

	if _, err := missing.Run(ctx); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("err = %v, want not exist", err)
	}

	broken := filepath.Join(t.TempDir(), "broken.ndjson")
	if err := os.WriteFile(broken, []byte("{\"user_id\":1}\nnot json\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := NewBootstrapService(topology, wallets, broken).Run(ctx); err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("err = %v, want error about line 2", err)
	}
}
