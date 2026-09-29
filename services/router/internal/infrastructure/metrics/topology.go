package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/router/internal/models"
)

type Topology struct {
	version    prometheus.Gauge
	keyShare   *prometheus.GaugeVec
	movedShare prometheus.Gauge
	ringPoints prometheus.Gauge
}

func NewTopology(reg prometheus.Registerer) *Topology {
	m := &Topology{
		version: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "router_topology_version",
			Help: "Topology version applied on the router.",
		}),
		keyShare: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "router_key_share",
			Help: "Expected share of users per node under the current strategy.",
		}, []string{"node"}),
		movedShare: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "router_moved_share",
			Help: "Share of users whose node changed with the last topology.",
		}),
		ringPoints: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "router_ring_points",
			Help: "Points on the ring, or the number of nodes for strategies without a ring.",
		}),
	}

	reg.MustRegister(m.version, m.keyShare, m.movedShare, m.ringPoints)

	return m
}

func (m *Topology) ObserveTopology(stats models.TopologyStats) {
	m.version.Set(float64(stats.Version))
	m.movedShare.Set(stats.MovedShare)
	m.ringPoints.Set(float64(stats.RingPoints))

	m.keyShare.Reset()

	for node, share := range stats.KeyShare {
		m.keyShare.WithLabelValues(node).Set(share)
	}
}
