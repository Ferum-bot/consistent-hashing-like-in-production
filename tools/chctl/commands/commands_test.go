package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/tools/chctl/client"
)

type fakeRouter struct {
	mu       sync.Mutex
	topology *client.Topology
	pending  int64
	applied  []client.Topology
	prepared []client.Topology
}

func (f *fakeRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch r.Method + " " + r.URL.Path {
	case "GET /healthz":
		w.WriteHeader(http.StatusOK)
	case "GET /topology":
		if f.topology == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"topology is not set"}`))

			return
		}

		current := *f.topology
		current.PendingVersion = f.pending

		_ = json.NewEncoder(w).Encode(current)
	case "POST /topology":
		var topology client.Topology
		_ = json.NewDecoder(r.Body).Decode(&topology)

		f.topology = &topology
		f.applied = append(f.applied, topology)

		_ = json.NewEncoder(w).Encode(topology)
	case "POST /topology/prepare":
		var topology client.Topology
		_ = json.NewDecoder(r.Body).Decode(&topology)

		f.prepared = append(f.prepared, topology)

		_ = json.NewEncoder(w).Encode(f.topology)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func startRouter(t *testing.T, router *fakeRouter) *client.Router {
	t.Helper()

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return client.NewRouter(server.URL, server.Client())
}

func testEnv(routers ...*client.Router) (Env, *bytes.Buffer) {
	out := &bytes.Buffer{}

	return Env{Clients: client.NewFactory(http.DefaultClient), Routers: routers, Out: out}, out
}

func writeFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "topology.json")

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	return path
}

const twoNodes = `"nodes": [{"id": "wallet-1", "addr": "http://wallet-1:8080"}, {"id": "wallet-2", "addr": "http://wallet-2:8080"}]`

func TestFind(t *testing.T) {
	tests := []struct {
		args   []string
		name   string
		rest   int
		reason string
	}{
		{args: []string{"status"}, name: "status"},
		{args: []string{"load", "set", "rps=1"}, name: "load set", rest: 1},
		{args: []string{"load"}, reason: `"load" needs a subcommand`},
		{args: []string{"nope"}, reason: `unknown command "nope"`},
		{args: nil, reason: ""},
	}

	for _, tt := range tests {
		command, rest, err := Find(tt.args)

		if tt.name != "" {
			if err != nil || command.Name != tt.name || len(rest) != tt.rest {
				t.Fatalf("Find(%v) = %q, %v, %v", tt.args, command.Name, rest, err)
			}

			continue
		}

		var usage UsageError
		if !errors.As(err, &usage) || usage.Reason != tt.reason {
			t.Fatalf("Find(%v) err = %v, want usage %q", tt.args, err, tt.reason)
		}
	}
}

func TestApply_GivesEveryRouterTheNextVersion(t *testing.T) {
	first := &fakeRouter{topology: &client.Topology{Version: 4, Strategy: "mod"}}
	second := &fakeRouter{topology: &client.Topology{Version: 3, Strategy: "mod"}}
	env, _ := testEnv(startRouter(t, first), startRouter(t, second))

	path := writeFile(t, `{"strategy": "ring", "vnodes": 16, `+twoNodes+`}`)

	if err := runApply(context.Background(), env, []string{path}); err != nil {
		t.Fatal(err)
	}

	for _, router := range []*fakeRouter{first, second} {
		if len(router.applied) != 1 || router.applied[0].Version != 5 {
			t.Fatalf("applied = %+v, want one topology with version 5", router.applied)
		}
	}
}

func TestApply_MigrationWithoutTopologyAppliesDirectly(t *testing.T) {
	router := &fakeRouter{}
	env, out := testEnv(startRouter(t, router))

	path := writeFile(t, `{"strategy": "ring", "migrate": true, `+twoNodes+`}`)

	if err := runApply(context.Background(), env, []string{path}); err != nil {
		t.Fatal(err)
	}

	if len(router.applied) != 1 || len(router.prepared) != 0 {
		t.Fatalf("applied %d, prepared %d: nothing to migrate, the topology must be applied directly", len(router.applied), len(router.prepared))
	}

	if !strings.Contains(out.String(), "nothing to migrate") {
		t.Fatalf("output must explain why migration was skipped:\n%s", out)
	}
}

func TestApply_RefusesUnfinishedMigration(t *testing.T) {
	router := &fakeRouter{topology: &client.Topology{Version: 2, Strategy: "ring"}, pending: 3}
	env, _ := testEnv(startRouter(t, router))

	path := writeFile(t, `{"strategy": "ring", `+twoNodes+`}`)

	err := runApply(context.Background(), env, []string{path})
	if err == nil || !strings.Contains(Explain(err), hintAbort) {
		t.Fatalf("err = %v, want a hint to abort", err)
	}

	if len(router.applied) != 0 {
		t.Fatal("nothing must be applied while a migration is unfinished")
	}
}

func TestReadTopologyFile_RejectsMistakes(t *testing.T) {
	tests := map[string]string{
		"unknown field": `{"strategy": "ring", "vnode": 16, ` + twoNodes + `}`,
		"version":       `{"version": 3, "strategy": "ring", ` + twoNodes + `}`,
		"no nodes":      `{"strategy": "ring"}`,
		"broken json":   `{"strategy": `,
	}

	for name, content := range tests {
		if _, err := readTopologyFile(writeFile(t, content)); err == nil {
			t.Fatalf("%s: file must be rejected", name)
		}
	}
}

func TestStatus_ShowsWhatIsDown(t *testing.T) {
	up := startRouter(t, &fakeRouter{topology: &client.Topology{Version: 7, Strategy: "mod", Nodes: []client.Node{{ID: "wallet-1"}}}})
	down := client.NewRouter("http://127.0.0.1:1", http.DefaultClient)

	env, out := testEnv(up, down)
	env.Loadgen = client.NewLoadgen("http://127.0.0.1:1", http.DefaultClient)

	if err := runStatus(context.Background(), env, nil); err != nil {
		t.Fatal(err)
	}

	report := out.String()

	for _, want := range []string{"v7  mod, 1 nodes", "down  nothing listens on this address"} {
		if !strings.Contains(report, want) {
			t.Fatalf("status must contain %q:\n%s", want, report)
		}
	}
}

func TestExplain_AddsHints(t *testing.T) {
	router := client.NewRouter("http://127.0.0.1:1", http.DefaultClient)

	_, _, err := router.Topology(context.Background())
	if !client.IsUnreachable(err) {
		t.Fatalf("err = %v, want unreachable", err)
	}

	if !strings.Contains(Explain(err), hintStartStand) {
		t.Fatalf("explain = %q, want a hint to start the stand", Explain(err))
	}

	wrapped := withHint(errors.New("boom"), "do this")
	if Explain(wrapped) != "boom\nhint: do this" {
		t.Fatalf("explain = %q", Explain(wrapped))
	}
}

func TestApply_RestartedRouterCatchesUpFirst(t *testing.T) {
	running := &fakeRouter{topology: &client.Topology{Version: 4, Strategy: "ring", VNodes: 16, MembershipDelay: map[string]string{"router-2": "30s"}}}
	restarted := &fakeRouter{}
	env, out := testEnv(startRouter(t, running), startRouter(t, restarted))

	path := writeFile(t, `{"strategy": "ring", "vnodes": 16, `+twoNodes+`}`)

	if err := runApply(context.Background(), env, []string{path}); err != nil {
		t.Fatal(err)
	}

	if len(restarted.applied) != 2 || restarted.applied[0].Version != 4 || restarted.applied[1].Version != 5 {
		t.Fatalf("restarted router must get v4 first and then v5, got %+v", restarted.applied)
	}

	if restarted.applied[0].MembershipDelay != nil {
		t.Fatal("catching up must not be delayed")
	}

	if !strings.Contains(out.String(), "has no topology") {
		t.Fatalf("output must explain the catch up:\n%s", out)
	}
}
