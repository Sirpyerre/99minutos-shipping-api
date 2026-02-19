package ports

import (
	"context"

	"github.com/99minutos/shipping-system/internal/core/domain"
)

// ShipmentRepository defines persistence operations for shipments.
type ShipmentRepository interface {
	Create(ctx context.Context, s *domain.Shipment) error
	// FindByTrackingNumber retrieves a shipment by tracking number.
	// When clientID is non-empty, the query is additionally filtered by client_id (for RBAC).
	FindByTrackingNumber(ctx context.Context, trackingNumber string, clientID string) (*domain.Shipment, error)
	FindByIdempotencyKey(ctx context.Context, key string) (*domain.Shipment, error)
}
