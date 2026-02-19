package ports

import (
	"context"
	"time"

	"github.com/99minutos/shipping-system/internal/core/domain"
)

// EventRepository handles event persistence and atomic shipment status updates.
type EventRepository interface {
	// UpdateShipmentStatus atomically sets the shipment's new status and appends
	// a history entry. The source string is stored as the entry notes.
	UpdateShipmentStatus(
		ctx context.Context,
		trackingNumber string,
		status domain.ShipmentStatus,
		ts time.Time,
		source string,
		location *domain.Coordinates,
	) error

	// InsertEvent persists an event to the status_events audit collection.
	InsertEvent(ctx context.Context, event *domain.TrackingEvent) error
}
