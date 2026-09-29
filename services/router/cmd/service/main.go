package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/service"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/router/internal/api"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/router/internal/clients"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/router/internal/infrastructure/http/client"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/router/internal/infrastructure/http/proxy"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/router/internal/infrastructure/metrics"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/router/internal/services"
)

const walletClientTimeout = 60 * time.Second

func main() {
	service.Main("router", run)
}

func run(ctx context.Context, configPath string) error {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return err
	}

	registry := service.NewRegistry()

	wallets := clients.NewWalletClient(&http.Client{Transport: client.NewTransport(), Timeout: walletClientTimeout})
	migrationMetrics := metrics.NewMigration(registry)

	topology := services.NewTopologyService(cfg.Name, cfg.Users, wallets, metrics.NewTopology(registry), migrationMetrics)
	routing := services.NewRoutingService(topology, migrationMetrics, cfg.GateTimeout)
	bootstrap := services.NewBootstrapService(topology, wallets, cfg.SnapshotPath)
	migration := services.NewMigrationService(topology, wallets, migrationMetrics)

	upstreamTransport := client.NewTransport()
	upstreamTransport.ResponseHeaderTimeout = cfg.UpstreamTimeout

	upstream := proxy.New(upstreamTransport, metrics.NewUpstream(registry))

	go topology.RunDeferred(ctx)

	slog.Info("router started", "addr", cfg.Addr, "name", cfg.Name, "users", cfg.Users, "snapshot", cfg.SnapshotPath)

	return service.Service{
		Name:     "router",
		Title:    "Router",
		Addr:     cfg.Addr,
		Spec:     api.OpenAPISpec,
		Registry: registry,
		Routes:   routes(routing, topology, bootstrap, migration, upstream),
	}.Run(ctx)
}

func routes(
	routing *services.RoutingService,
	topology *services.TopologyService,
	bootstrap *services.BootstrapService,
	migration *services.MigrationService,
	upstream *proxy.Proxy,
) func(mux *http.ServeMux) {
	walletAPI := api.NewWalletAPI(routing, upstream)
	locateAPI := api.NewLocateAPI(routing)
	topologyAPI := api.NewTopologyAPI(topology)
	bootstrapAPI := api.NewBootstrapAPI(bootstrap)
	migrationAPI := api.NewMigrationAPI(migration)

	return func(mux *http.ServeMux) {
		mux.HandleFunc("GET /wallet/{userID}", walletAPI.Read)
		mux.HandleFunc("POST /wallet/{userID}/deposit", walletAPI.Write)
		mux.HandleFunc("POST /wallet/{userID}/withdraw", walletAPI.Write)

		mux.HandleFunc("GET /locate/{userID}", locateAPI.Get)

		mux.HandleFunc("GET /topology", topologyAPI.Get)
		mux.HandleFunc("POST /topology", topologyAPI.Apply)
		mux.HandleFunc("POST /topology/prepare", topologyAPI.Prepare)
		mux.HandleFunc("POST /topology/commit", topologyAPI.Commit)
		mux.HandleFunc("POST /topology/abort", topologyAPI.Abort)

		mux.HandleFunc("POST /migrate", migrationAPI.Run)
		mux.HandleFunc("POST /migrate/cleanup", migrationAPI.Cleanup)

		mux.HandleFunc("POST /bootstrap", bootstrapAPI.Run)
	}
}
