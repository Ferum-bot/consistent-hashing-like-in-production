package middlewares

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/http/recorder"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/metrics"
)

func WithMetrics(reg prometheus.Registerer, namespace string) func(http.Handler) http.Handler {
	requests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "http_requests_total",
		Help:      "HTTP requests by route and status code.",
	}, []string{"route", "code"})

	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request latency by route.",
		Buckets:   metrics.LatencyBuckets,
	}, []string{"route"})

	reg.MustRegister(requests, duration)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := recorder.New(w)

			next.ServeHTTP(rec, r)

			route := r.Pattern
			if route == "" {
				route = "unmatched"
			}

			requests.WithLabelValues(route, strconv.Itoa(rec.Status())).Inc()
			duration.WithLabelValues(route).Observe(time.Since(start).Seconds())
		})
	}
}
