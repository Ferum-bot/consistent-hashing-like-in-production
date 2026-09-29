package metrics

import "github.com/prometheus/client_golang/prometheus"

var LatencyBuckets = prometheus.ExponentialBuckets(0.0001, 2, 16)
