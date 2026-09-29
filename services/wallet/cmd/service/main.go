package main

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/service"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/wallet/internal/api"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/wallet/internal/infrastructure/http/middlewares"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/wallet/internal/infrastructure/metrics"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/wallet/internal/repositories"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/wallet/internal/services"
)

func main() {
	service.Main("wallet", run)
}

func run(ctx context.Context, configPath string) error {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return err
	}

	repo := repositories.NewMemoryRepository()
	wallets := services.NewWalletService(repo)
	fence := services.NewTopologyFence()

	registry := service.NewRegistry()
	metrics.RegisterNode(registry, wallets, fence)

	slog.Info("wallet started", "addr", cfg.Addr, "node", cfg.NodeID)

	return service.Service{
		Name:     "wallet",
		Title:    "Wallet",
		Addr:     cfg.Addr,
		Spec:     api.OpenAPISpec,
		Registry: registry,
		Routes:   routes(cfg.NodeID, wallets, fence),
	}.Run(ctx)
}

func routes(nodeID string, wallets *services.WalletService, fence *services.TopologyFence) func(mux *http.ServeMux) {
	walletAPI := api.NewWalletAPI(wallets)
	migrationAPI := api.NewMigrationAPI(wallets)
	statsAPI := api.NewStatsAPI(nodeID, wallets, fence)
	fenceAPI := api.NewFenceAPI(fence)

	fenced := middlewares.WithTopologyFence(fence)

	return func(mux *http.ServeMux) {
		mux.Handle("GET /wallet/{userID}", fenced(http.HandlerFunc(walletAPI.Get)))
		mux.Handle("POST /wallet/{userID}/deposit", fenced(http.HandlerFunc(walletAPI.Deposit)))
		mux.Handle("POST /wallet/{userID}/withdraw", fenced(http.HandlerFunc(walletAPI.Withdraw)))

		mux.HandleFunc("GET /export", migrationAPI.Export)
		mux.Handle("POST /import", fenced(http.HandlerFunc(migrationAPI.Import)))
		mux.Handle("POST /evict", fenced(http.HandlerFunc(migrationAPI.Evict)))

		mux.Handle("POST /fence", fenced(http.HandlerFunc(fenceAPI.Announce)))

		mux.HandleFunc("GET /stats", statsAPI.Get)
	}
}
