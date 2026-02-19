package queue

import (
	"context"
	"hash/fnv"

	"github.com/rs/zerolog"

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
	d.workers[d.shardIndex(event.TrackingNumber)] <- event
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
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			if err := d.service.Process(ctx, event); err != nil {
				d.log.Error().Err(err).
					Str("tracking_number", event.TrackingNumber).
					Int("worker_id", id).
					Msg("event processing failed")
			}
		}
	}
}
