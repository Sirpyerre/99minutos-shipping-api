package ports

import (
	"context"
	"time"

	"github.com/99minutos/shipping-system/internal/core/domain"
)

// ListShipmentsFilter carries all query parameters for listing shipments.
// ClientID is always enforced by the service layer (RBAC).
type ListShipmentsFilter struct {
	ClientID    string    // empty = no filter (admin); non-empty = scoped to client
	Status      string    // optional: filter by shipment status
	ServiceType string    // optional: filter by service type
	Search      string    // optional: partial match on tracking_number or sender.name
	DateFrom    time.Time // optional: created_at >= DateFrom
	DateTo      time.Time // optional: created_at <= DateTo
	Page        int       // 1-based
	Limit       int       // max rows per page (capped at 100 by service)
}

// ShipmentRepository defines persistence operations for shipments.
type ShipmentRepository interface {
	Create(ctx context.Context, s *domain.Shipment) error
	// FindByTrackingNumber retrieves a shipment by tracking number.
	// When clientID is non-empty, the query is additionally filtered by client_id (for RBAC).
	FindByTrackingNumber(ctx context.Context, trackingNumber string, clientID string) (*domain.Shipment, error)
	FindByIdempotencyKey(ctx context.Context, key string) (*domain.Shipment, error)
	// List returns a page of shipments matching filter and the total count.
	List(ctx context.Context, filter ListShipmentsFilter) ([]*domain.Shipment, int64, error)
}
