package queue

import (
	"context"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/rs/zerolog"

	apimetrics "github.com/99minutos/shipping-system/internal/api/metrics"
	"github.com/99minutos/shipping-system/internal/core/ports"
)

const (
	defaultWorkers = 8
	channelBuffer  = 256
)

// Dispatcher routes tracking events to a fixed set of workers using consistent
// hashing on the tracking number, guaranteeing per-shipment event ordering.
type Dispatcher struct {
	workers []chan ports.TrackingEventInput
	service ports.EventService
	log     zerolog.Logger
}

// NewDispatcher creates a Dispatcher with numWorkers sharded workers.
// If numWorkers <= 0, defaultWorkers is used.
func NewDispatcher(numWorkers int, service ports.EventService, log zerolog.Logger) *Dispatcher {
	if numWorkers <= 0 {
		numWorkers = defaultWorkers
	}
	d := &Dispatcher{
		workers: make([]chan ports.TrackingEventInput, numWorkers),
		service: service,
		log:     log,
	}
	for i := range d.workers {
		d.workers[i] = make(chan ports.TrackingEventInput, channelBuffer)
	}
	return d
}

// Start launches all worker goroutines. Workers stop when ctx is cancelled.
func (d *Dispatcher) Start(ctx context.Context) {
	for i, ch := range d.workers {
		go d.runWorker(ctx, i, ch)
	}
}

// Enqueue sends an event to the worker responsible for its tracking number.
// The call is non-blocking up to channelBuffer capacity.
func (d *Dispatcher) Enqueue(event ports.TrackingEventInput) {
	idx := d.shardIndex(event.TrackingNumber)
	d.workers[idx] <- event
	apimetrics.EventsQueueDepth.WithLabelValues(fmt.Sprintf("%d", idx)).Set(float64(len(d.workers[idx])))
}

// EnqueueBatch enqueues multiple events preserving per-shipment ordering.
func (d *Dispatcher) EnqueueBatch(events []ports.TrackingEventInput) {
	for _, e := range events {
		d.Enqueue(e)
	}
}

// shardIndex maps a tracking number deterministically to a worker index.
func (d *Dispatcher) shardIndex(trackingNumber string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(trackingNumber))
	return int(h.Sum32()) % len(d.workers)
}

func (d *Dispatcher) runWorker(ctx context.Context, id int, ch <-chan ports.TrackingEventInput) {
	workerLabel := fmt.Sprintf("%d", id)
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			// Update queue depth after dequeue
			apimetrics.EventsQueueDepth.WithLabelValues(workerLabel).Set(float64(len(ch)))

			start := time.Now()
			err := d.service.Process(ctx, event)
			elapsed := time.Since(start).Seconds()

			statusLabel := event.Status
			if err != nil {
				statusLabel = "error"
				d.log.Error().Err(err).
					Str("tracking_number", event.TrackingNumber).
					Int("worker_id", id).
					Msg("event processing failed")
			}
			apimetrics.EventProcessingDuration.WithLabelValues(statusLabel).Observe(elapsed)
		}
	}
}
