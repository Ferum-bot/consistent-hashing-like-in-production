package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/loadgen/internal/models"
)

type Loadgen struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	dropped  prometheus.Counter
	checks   *prometheus.CounterVec
	profile  *prometheus.GaugeVec
}

func NewLoadgen(reg prometheus.Registerer, uncertainUsers func() int) *Loadgen {
	m := &Loadgen{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "loadgen_requests_total",
			Help: "Operations sent to routers by kind, response status and router.",
		}, []string{"op", "code", "router"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "loadgen_request_duration_seconds",
			Help:    "Time from the moment an operation was due to its response, by kind.",
			Buckets: prometheus.ExponentialBuckets(0.0001, 2, 18),
		}, []string{"op"}),
		dropped: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "loadgen_dropped_total",
			Help: "Operations that were due but not sent because workers could not keep up.",
		}),
		checks: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "loadgen_balance_checks_total",
			Help: "Balance checks against the expected ledger by result.",
		}, []string{"result"}),
		profile: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "loadgen_profile",
			Help: "Current load profile parameters.",
		}, []string{"param"}),
	}

	uncertain := prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "loadgen_uncertain_users",
		Help: "Users whose last write ended without a clear answer and are excluded from checks.",
	}, func() float64 {
		return float64(uncertainUsers())
	})

	reg.MustRegister(m.requests, m.duration, m.dropped, m.checks, m.profile, uncertain)

	return m
}

func (m *Loadgen) ObserveRequest(kind models.OperationKind, router string, status int, duration time.Duration) {
	code := "error"

	if status != 0 {
		code = strconv.Itoa(status)
	}

	m.requests.WithLabelValues(string(kind), code, router).Inc()
	m.duration.WithLabelValues(string(kind)).Observe(duration.Seconds())
}

func (m *Loadgen) ObserveDropped() {
	m.dropped.Inc()
}

func (m *Loadgen) ObserveCheck(result models.CheckResult) {
	m.checks.WithLabelValues(string(result)).Inc()
}

func (m *Loadgen) ObserveProfile(profile models.Profile) {
	rps := 0.0

	if profile.Running {
		rps = float64(profile.RPS)
	}

	m.profile.WithLabelValues("target_rps").Set(rps)
	m.profile.WithLabelValues("read_ratio").Set(profile.ReadRatio)
	m.profile.WithLabelValues("deposit_share").Set(profile.DepositShare)
	m.profile.WithLabelValues("read_zipf_s").Set(profile.ReadZipfS)
	m.profile.WithLabelValues("write_zipf_s").Set(profile.WriteZipfS)
	m.profile.WithLabelValues("hot_share").Set(profile.HotShare)
	m.profile.WithLabelValues("verify_rps").Set(float64(profile.VerifyRPS))
}
