package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/service"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/loadgen/internal/api"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/loadgen/internal/clients"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/loadgen/internal/infrastructure/metrics"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/loadgen/internal/repositories"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/loadgen/internal/services"
)

const maxIdleConnsPerHost = 512

func main() {
	service.Main("loadgen", run)
}

func run(ctx context.Context, configPath string) error {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return err
	}

	registry := service.NewRegistry()

	ledgerRepo := repositories.NewLedgerRepository()
	observer := metrics.NewLoadgen(registry, ledgerRepo.UncertainUsers)

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConnsPerHost = maxIdleConnsPerHost

	routers := clients.NewRouterClient(&http.Client{Transport: transport, Timeout: cfg.RequestTimeout})
	checker := services.NewChecker(ledgerRepo, observer)
	ledger := services.NewLedgerService(ledgerRepo, cfg.SnapshotPath)

	generator, err := services.NewGenerator(
		ledgerRepo, routers, observer, checker, cfg.Targets, cfg.Workers, cfg.Profile.toModel(),
	)
	if err != nil {
		return err
	}

	go loadLedger(ctx, ledger)
	go generator.Run(ctx)

	slog.Info("loadgen started", "addr", cfg.Addr, "targets", cfg.Targets, "snapshot", cfg.SnapshotPath)

	return service.Service{
		Name:     "loadgen",
		Title:    "Loadgen",
		Addr:     cfg.Addr,
		Spec:     api.OpenAPISpec,
		Registry: registry,
		Routes:   routes(generator, ledger),
	}.Run(ctx)
}

func loadLedger(ctx context.Context, ledger *services.LedgerService) {
	start := time.Now()

	stats, err := ledger.Reload(ctx)
	if err != nil {
		slog.Warn("ledger is not loaded, call POST /ledger/reload when the snapshot is ready", "err", err)
		return
	}

	slog.Info("ledger loaded", "users", stats.Users, "wallets", stats.Wallets, "took", time.Since(start).Round(time.Millisecond))
}

func routes(generator *services.Generator, ledger *services.LedgerService) func(mux *http.ServeMux) {
	configAPI := api.NewConfigAPI(generator)
	ledgerAPI := api.NewLedgerAPI(ledger)

	return func(mux *http.ServeMux) {
		mux.HandleFunc("GET /config", configAPI.Get)
		mux.HandleFunc("PATCH /config", configAPI.Patch)

		mux.HandleFunc("GET /ledger", ledgerAPI.Get)
		mux.HandleFunc("POST /ledger/reload", ledgerAPI.Reload)
	}
}
