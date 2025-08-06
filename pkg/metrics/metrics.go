package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	WebhookReceived = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "webhook_events_received_total",
		Help: "The total number of webhook events received",
	}, []string{"client_id", "event_type"})

	WebhookProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "webhook_events_processed_total",
		Help: "The total number of webhook events processed",
	}, []string{"client_id", "event_type", "status"})

	WebhookProcessingTime = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "webhook_processing_duration_seconds",
		Help:    "Time taken to process webhook events",
		Buckets: prometheus.DefBuckets,
	}, []string{"client_id", "event_type"})

	WebhookQueueSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "webhook_queue_size",
		Help: "Current size of the webhook processing queue",
	}, []string{"client_id"})

	WebhookRetries = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "webhook_retries_total",
		Help: "The total number of webhook event retries",
	}, []string{"client_id", "event_type"})

	RateLimitExceeded = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "webhook_rate_limit_exceeded_total",
		Help: "The total number of times rate limits were exceeded",
	}, []string{"client_id", "limit_type"})
)
