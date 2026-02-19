// Package metrics defines and registers all custom Prometheus metrics for the
// 99minutos shipping API. It is the single source of truth for metric names,
// labels, and help strings.
//
// Call Register() once at startup (before the HTTP server starts) to register
// all metrics with the default Prometheus registry.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const namespace = "shipping"

// ── Event metrics ─────────────────────────────────────────────────────────────

// EventsProcessedTotal counts events that completed processing successfully.
// Labels:
//   - status: the new shipment status applied by the event (e.g. "picked_up")
//   - source: the event source reported by the sender (e.g. "driver_app")
var EventsProcessedTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "events_processed_total",
		Help:      "Total number of tracking events successfully processed.",
	},
	[]string{"status", "source"},
)

// EventsErrorsTotal counts events that failed processing.
// Label:
//   - reason: short description of the failure (e.g. "invalid_transition", "shipment_not_found", "update_failed")
var EventsErrorsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "events_errors_total",
		Help:      "Total number of tracking events that failed processing.",
	},
	[]string{"reason"},
)

// EventsDedupTotal counts deduplication decisions.
// Label:
//   - result: "hit" (duplicate, skipped) or "miss" (new event, processed)
var EventsDedupTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "events_dedup_total",
		Help:      "Total number of deduplication checks, labelled by result (hit/miss).",
	},
	[]string{"result"},
)

// EventsQueueDepth tracks the current number of events waiting in each worker channel.
// Label:
//   - worker_id: numeric worker index (e.g. "0", "1", …)
var EventsQueueDepth = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "events_queue_depth",
		Help:      "Current number of events pending in each dispatcher worker channel.",
	},
	[]string{"worker_id"},
)

// EventProcessingDuration measures how long a single event takes to process end-to-end.
// Label:
//   - status: the resulting shipment status, or "error" on failure
var EventProcessingDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "event_processing_duration_seconds",
		Help:      "Duration of event processing from dequeue to persistence.",
		Buckets:   prometheus.DefBuckets, // .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10
	},
	[]string{"status"},
)

// ── Shipment metrics ──────────────────────────────────────────────────────────

// ShipmentsCreatedTotal counts newly created shipments.
// Label:
//   - service_type: "same_day", "next_day", or "standard"
var ShipmentsCreatedTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "shipments_created_total",
		Help:      "Total number of shipments created, by service type.",
	},
	[]string{"service_type"},
)
