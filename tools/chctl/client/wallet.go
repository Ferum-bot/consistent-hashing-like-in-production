package client

import (
	"context"
	"net/http"
	"time"
)

const (
	importTimeout     = time.Minute
	ndjsonContentType = "application/x-ndjson"
)

type WalletStats struct {
	Node            string `json:"node"`
	TopologyVersion int64  `json:"topology_version"`
	Strict          bool   `json:"strict"`
	Users           int    `json:"users"`
	Wallets         int    `json:"wallets"`
}

type Wallet struct {
	endpoint
}

func NewWallet(url string, client *http.Client) *Wallet {
	return &Wallet{endpoint: newEndpoint("wallet", url, client)}
}

func (w *Wallet) Stats(ctx context.Context) (WalletStats, error) {
	var stats WalletStats

	err := w.get(ctx, "/stats", &stats)

	return stats, err
}

func (w *Wallet) Import(ctx context.Context, users []byte) (int, error) {
	var imported struct {
		Imported int `json:"imported"`
	}

	req := request{
		method:      http.MethodPost,
		path:        "/import",
		body:        users,
		contentType: ndjsonContentType,
		timeout:     importTimeout,
	}

	err := w.do(ctx, req, &imported)

	return imported.Imported, err
}
