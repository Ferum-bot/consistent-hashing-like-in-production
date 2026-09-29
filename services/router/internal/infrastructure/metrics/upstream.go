package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	libmetrics "github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/metrics"
)

type Upstream struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

func NewUpstream(reg prometheus.Registerer) *Upstream {
	m := &Upstream{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "router_upstream_requests_total",
			Help: "Requests forwarded to wallet nodes by node and status code.",
		}, []string{"node", "code"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "router_upstream_request_duration_seconds",
			Help:    "Latency of requests forwarded to wallet nodes by node.",
			Buckets: libmetrics.LatencyBuckets,
		}, []string{"node"}),
	}

	reg.MustRegister(m.requests, m.duration)

	return m
}

func (m *Upstream) ObserveUpstream(node string, code int, duration time.Duration) {
	m.requests.WithLabelValues(node, strconv.Itoa(code)).Inc()
	m.duration.WithLabelValues(node).Observe(duration.Seconds())
}
