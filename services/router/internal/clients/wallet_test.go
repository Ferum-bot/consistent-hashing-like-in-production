package clients

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/headers"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/sharding"
)

func TestWalletClient_RetriesBrokenConnection(t *testing.T) {
	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				_ = conn.(*net.TCPConn).SetLinger(0)
				conn.Close()
			}

			return
		}

		if r.Header.Get(headers.TopologyVersion) != "7" {
			t.Errorf("version header = %q, want 7", r.Header.Get(headers.TopologyVersion))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewWalletClient(server.Client())
	node := sharding.Node{ID: "wallet-1", Addr: server.URL}

	if err := client.Import(context.Background(), node, 7, []byte(`{"user_id":1,"wallets":[]}`+"\n")); err != nil {
		t.Fatal(err)
	}

	if calls.Load() != 2 {
		t.Fatalf("calls = %d, want a retry after the broken connection", calls.Load())
	}
}

func TestWalletClient_DropsDeadPooledConnections(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewWalletClient(&http.Client{Transport: &http.Transport{MaxIdleConnsPerHost: 16}})
	node := sharding.Node{ID: "wallet-1", Addr: server.URL}

	for range 8 {
		if err := client.Import(context.Background(), node, 0, []byte("\n")); err != nil {
			t.Fatal(err)
		}
	}

	server.CloseClientConnections()

	if err := client.Import(context.Background(), node, 0, []byte("\n")); err != nil {
		t.Fatalf("import after the node dropped every pooled connection: %v", err)
	}
}
