package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	stockClientTimer = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "stockticker",
			Subsystem: "stock_controller",
			Name:      "stock_request_duration_seconds",
			Help:      "Bucketed histogram of stock request timings",

			// 20ms to 33s. See: https://go.dev/play/p/XpPPmtYsLLD
			Buckets: prometheus.ExponentialBuckets(.2, 1.9, 9),
		},
		[]string{"resolution"},
	)

	stockClientErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "stockticker",
			Subsystem: "stock_controller",
			Name:      "stock_client_errors_total",
			Help:      "Number of errors from the stock client",
		},
		[]string{"resolution"},
	)

	stockCacheTimer = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "stockticker",
			Subsystem: "stock_controller",
			Name:      "stock_cache_duration_seconds",
			Help:      "Bucketed histogram of stock cache access timings",

			// 20ms to 33s. See: https://go.dev/play/p/XpPPmtYsLLD
			Buckets: prometheus.ExponentialBuckets(.2, 1.9, 9),
		},
		[]string{"operation"},
	)

	stockCacheErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "stockticker",
			Subsystem: "stock_controller",
			Name:      "stock_cache_errors_total",
			Help:      "Number of errors from the stock cache client",
		},
		[]string{"operation"},
	)
)
