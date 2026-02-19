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

// StatusHistoryItem is a single entry in the shipment's status history.
type StatusHistoryItem struct {
	Status    string
	Timestamp time.Time
	Notes     string
}

// ShipmentDetail is the full shipment view returned by GetShipment.
type ShipmentDetail struct {
	TrackingNumber    string
	Status            string
	ServiceType       string
	CreatedAt         time.Time
	EstimatedDelivery time.Time
	Sender            SenderInput
	Origin            AddressInput
	Destination       AddressInput
	Package           PackageInput
	StatusHistory     []StatusHistoryItem
}

// ShipmentService defines use-case operations for shipments.
type ShipmentService interface {
	CreateShipment(ctx context.Context, input CreateShipmentInput) (*ShipmentResult, error)
	GetShipment(ctx context.Context, input GetShipmentInput) (*ShipmentDetail, error)
	ListShipments(ctx context.Context, input ListShipmentsInput) (*ListShipmentsResult, error)
}

// ListShipmentsInput carries all parameters for the list endpoint.
type ListShipmentsInput struct {
	Role        string
	ClientID    string
	Status      string
	ServiceType string
	Search      string
	DateFrom    time.Time
	DateTo      time.Time
	Page        int
	Limit       int
}

// ShipmentSummary is the lightweight view used in list responses (no status_history).
type ShipmentSummary struct {
	TrackingNumber    string
	Status            string
	ServiceType       string
	ClientID          string
	Sender            SenderInput
	Origin            AddressInput
	Destination       AddressInput
	CreatedAt         time.Time
	EstimatedDelivery time.Time
}

// ListShipmentsResult is returned by ListShipments.
type ListShipmentsResult struct {
	Items      []ShipmentSummary
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}
