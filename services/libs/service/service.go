package service

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/middlewares"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/server"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/swagger"
)

type Service struct {
	Name     string
	Title    string
	Addr     string
	Spec     []byte
	Registry *prometheus.Registry
	Routes   func(mux *http.ServeMux)
}

func Main(name string, run func(ctx context.Context, configPath string) error) {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	go func() {
		<-ctx.Done()
		stop()
	}()

	err := run(ctx, *configPath)

	stop()

	if err != nil {
		slog.Error(name+" stopped", "err", err)
		os.Exit(1)
	}
}

func NewRegistry() *prometheus.Registry {
	registry := prometheus.NewRegistry()

	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	return registry
}

func (s Service) Run(ctx context.Context) error {
	return server.New(s.Addr, s.handler()).Run(ctx)
}

func (s Service) handler() http.Handler {
	docs := swagger.New(s.Title, s.Spec)
	mux := http.NewServeMux()

	s.Routes(mux)

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("GET /metrics", promhttp.HandlerFor(s.Registry, promhttp.HandlerOpts{}))

	mux.HandleFunc("GET /{$}", docs.RedirectToUI)
	mux.HandleFunc("GET "+swagger.UIPath, docs.UI)
	mux.HandleFunc("GET "+swagger.SpecPath, docs.Spec)

	return middlewares.WithMetrics(s.Registry, s.Name)(mux)
}
