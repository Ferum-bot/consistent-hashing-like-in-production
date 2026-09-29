package metrics

import (
	"context"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/wallet/internal/models"
)

func RegisterNode(reg prometheus.Registerer, stats StatsService, fence TopologyFence) {
	users := prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "wallet_users",
		Help: "Users stored on the node.",
	}, func() float64 {
		return float64(currentStats(stats).Users)
	})

	wallets := prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "wallet_wallets",
		Help: "Wallets stored on the node.",
	}, func() float64 {
		return float64(currentStats(stats).Wallets)
	})

	version := prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "wallet_topology_version",
		Help: "Newest topology version the node has seen.",
	}, func() float64 {
		return float64(fence.Current())
	})

	strict := prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "wallet_fence_strict",
		Help: "1 when the node rejects requests without a topology version.",
	}, func() float64 {
		if fence.Strict() {
			return 1
		}

		return 0
	})

	reg.MustRegister(users, wallets, version, strict)
}

func currentStats(stats StatsService) models.Stats {
	s, err := stats.Stats(context.Background())
	if err != nil {
		slog.Warn("collect stats", "err", err)
		return models.Stats{}
	}

	return s
}
