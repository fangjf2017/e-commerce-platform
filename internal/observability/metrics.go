package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	FlagEvaluationsTotal    *prometheus.CounterVec
	EvaluationDuration      *prometheus.HistogramVec
	CacheHitsTotal          *prometheus.CounterVec
	CacheMissesTotal        *prometheus.CounterVec
	CacheInvalidationsTotal *prometheus.CounterVec
	FlagUpdatesTotal        *prometheus.CounterVec
	ActiveFlagsGauge        *prometheus.GaugeVec
	HTTPRequestsTotal       *prometheus.CounterVec
	HTTPRequestDuration     *prometheus.HistogramVec
}

func NewMetrics() *Metrics {
	return &Metrics{
		FlagEvaluationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "fms",
				Name:      "flag_evaluations_total",
				Help:      "Total number of flag evaluations.",
			},
			[]string{"flag_key", "environment", "result", "cache_layer"},
		),

		EvaluationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "fms",
				Name:      "evaluation_duration_seconds",
				Help:      "Duration of flag evaluations in seconds.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"flag_key", "environment"},
		),

		CacheHitsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "fms",
				Name:      "cache_hits_total",
				Help:      "Total number of cache hits.",
			},
			[]string{"tier"},
		),

		CacheMissesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "fms",
				Name:      "cache_misses_total",
				Help:      "Total number of cache misses.",
			},
			[]string{"tier"},
		),

		CacheInvalidationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "fms",
				Name:      "cache_invalidations_total",
				Help:      "Total number of cache invalidations.",
			},
			[]string{"reason"},
		),

		FlagUpdatesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "fms",
				Name:      "flag_updates_total",
				Help:      "Total number of flag updates.",
			},
			[]string{"action"},
		),

		ActiveFlagsGauge: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "fms",
				Name:      "active_flags",
				Help:      "Number of active flags by environment and status.",
			},
			[]string{"environment", "status"},
		),

		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "fms",
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests.",
			},
			[]string{"method", "path", "status_code"},
		),

		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "fms",
				Name:      "http_request_duration_seconds",
				Help:      "Duration of HTTP requests in seconds.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
	}
}
