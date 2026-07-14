package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var Registry = prometheus.NewRegistry()

func init() {
	Registry.MustRegister(
		WSReconnectCount,
		WSGapCount,
		WSBookLagSeconds,
		RateLimitHeadroom,
		FillLatencySeconds,
		BalanceSyncLagSeconds,
		EventsPublishedTotal,
		AuditEventsEmittedTotal,
		AuditDroppedTotal,
	)
}

var (
	WSReconnectCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ws_reconnect_count",
		Help: "Number of WebSocket reconnects per venue.",
	}, []string{"venue"})

	WSGapCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ws_gap_count",
		Help: "Number of detected sequence gaps per venue.",
	}, []string{"venue"})

	WSBookLagSeconds = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ws_book_lag_seconds",
		Help: "Lag between book update event timestamp and processing time.",
	}, []string{"venue", "pair"})

	RateLimitHeadroom = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "rate_limit_headroom",
		Help: "Remaining rate-limit weight budget per venue.",
	}, []string{"venue"})

	FillLatencySeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "fill_latency_seconds",
		Help:    "Latency between fill execution and event emission.",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 12),
	}, []string{"venue"})

	BalanceSyncLagSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "balance_sync_lag_seconds",
		Help:    "Lag between balance snapshot timestamp and processing time.",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 12),
	}, []string{"venue"})

	EventsPublishedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "events_published_total",
		Help: "Total events published to the event bus.",
	}, []string{"venue", "type", "status"})

	AuditEventsEmittedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "audit_events_emitted_total",
		Help: "Total audit events successfully emitted.",
	}, []string{"venue", "type"})

	AuditDroppedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "audit_dropped_total",
		Help: "Audit events dropped due to queue overflow.",
	}, []string{"venue", "type"})
)