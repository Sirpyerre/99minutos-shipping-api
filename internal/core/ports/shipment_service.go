package ports

import (
	"context"
	"time"
)

// CreateShipmentInput carries all data needed to create a new shipment.
type CreateShipmentInput struct {
	Sender         SenderInput
	Origin         AddressInput
	Destination    AddressInput
	Package        PackageInput
	ServiceType    string
	ClientID       string
	IdempotencyKey string
}

// SenderInput holds sender contact details.
type SenderInput struct {
	Name  string
	Email string
	Phone string
}

// CoordinatesInput holds geographic coordinates.
type CoordinatesInput struct {
	Lat float64
	Lng float64
}

// AddressInput holds a physical location.
type AddressInput struct {
	Address     string
	City        string
	ZipCode     string
	Coordinates CoordinatesInput
}

// DimensionsInput holds package size.
type DimensionsInput struct {
	LengthCm float64
	WidthCm  float64
	HeightCm float64
}

// PackageInput holds package details.
type PackageInput struct {
	WeightKg      float64
	Dimensions    DimensionsInput
	Description   string
	DeclaredValue float64
	Currency      string
}

// ShipmentResult is returned by the service after creating a shipment.
type ShipmentResult struct {
	TrackingNumber    string
	Status            string
	CreatedAt         time.Time
	EstimatedDelivery time.Time
	// AlreadyExisted is true when the Idempotency-Key matched an existing shipment.
	AlreadyExisted bool
}

// GetShipmentInput carries the parameters needed to retrieve a single shipment.
type GetShipmentInput struct {
	TrackingNumber string
	// Role and ClientID are used to enforce RBAC: "client" role only sees own shipments.
	Role     string
	ClientID string
}

// StatusHistoryItem is a single entry in the shipment's status history response.
type StatusHistoryItem struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Notes     string `json:"notes,omitempty"`
}

// ShipmentDetail is the full shipment view returned by GetShipment.
type ShipmentDetail struct {
	TrackingNumber    string              `json:"tracking_number"`
	Status            string              `json:"status"`
	ServiceType       string              `json:"service_type"`
	CreatedAt         string              `json:"created_at"`
	EstimatedDelivery string              `json:"estimated_delivery"`
	Sender            SenderInput         `json:"sender"`
	Origin            AddressInput        `json:"origin"`
	Destination       AddressInput        `json:"destination"`
	Package           PackageInput        `json:"package"`
	StatusHistory     []StatusHistoryItem `json:"status_history"`
}

// ShipmentService defines use-case operations for shipments.
type ShipmentService interface {
	CreateShipment(ctx context.Context, input CreateShipmentInput) (*ShipmentResult, error)
	GetShipment(ctx context.Context, input GetShipmentInput) (*ShipmentDetail, error)
}
