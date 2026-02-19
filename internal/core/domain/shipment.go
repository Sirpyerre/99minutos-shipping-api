package domain

import (
	"errors"
	"time"
)

// ShipmentStatus represents the lifecycle state of a shipment.
type ShipmentStatus string

const (
	StatusCreated     ShipmentStatus = "created"
	StatusPickedUp    ShipmentStatus = "picked_up"
	StatusInWarehouse ShipmentStatus = "in_warehouse"
	StatusInTransit   ShipmentStatus = "in_transit"
	StatusDelivered   ShipmentStatus = "delivered"
	StatusCancelled   ShipmentStatus = "cancelled"
)

// validTransitions defines the allowed state machine transitions.
var validTransitions = map[ShipmentStatus][]ShipmentStatus{
	StatusCreated:     {StatusPickedUp, StatusCancelled},
	StatusPickedUp:    {StatusInWarehouse, StatusCancelled},
	StatusInWarehouse: {StatusInTransit, StatusCancelled},
	StatusInTransit:   {StatusDelivered},
}

var ErrInvalidTransition = errors.New("invalid status transition")
var ErrShipmentNotFound = errors.New("shipment not found")
var ErrDuplicateShipment = errors.New("shipment already exists")
var ErrForbidden = errors.New("access forbidden")

// CanTransitionTo reports whether a transition from current status to next is valid.
func (s ShipmentStatus) CanTransitionTo(next ShipmentStatus) bool {
	for _, allowed := range validTransitions[s] {
		if allowed == next {
			return true
		}
	}
	return false
}

// Coordinates represents a geographic point.
type Coordinates struct {
	Lat float64 `json:"lat" bson:"lat"`
	Lng float64 `json:"lng" bson:"lng"`
}

// Address represents a physical location.
type Address struct {
	Address     string      `json:"address" bson:"address"`
	City        string      `json:"city" bson:"city"`
	ZipCode     string      `json:"zip_code" bson:"zip_code"`
	Coordinates Coordinates `json:"coordinates" bson:"coordinates"`
}

// Person represents a sender or recipient.
type Person struct {
	Name  string `json:"name" bson:"name"`
	Email string `json:"email" bson:"email"`
	Phone string `json:"phone" bson:"phone"`
}

// Dimensions represents the physical size of a package.
type Dimensions struct {
	LengthCm float64 `json:"length_cm" bson:"length_cm"`
	WidthCm  float64 `json:"width_cm" bson:"width_cm"`
	HeightCm float64 `json:"height_cm" bson:"height_cm"`
}

// Package contains the details of what is being shipped.
type Package struct {
	WeightKg      float64    `json:"weight_kg" bson:"weight_kg"`
	Dimensions    Dimensions `json:"dimensions" bson:"dimensions"`
	Description   string     `json:"description" bson:"description"`
	DeclaredValue float64    `json:"declared_value" bson:"declared_value"`
	Currency      string     `json:"currency" bson:"currency"`
}

// StatusHistoryEntry records a single status transition on a shipment.
type StatusHistoryEntry struct {
	Status    ShipmentStatus `json:"status" bson:"status"`
	Timestamp time.Time      `json:"timestamp" bson:"timestamp"`
	Notes     string         `json:"notes,omitempty" bson:"notes,omitempty"`
}

// Shipment is the core aggregate root.
type Shipment struct {
	ID                string         `json:"id" bson:"_id,omitempty"`
	TrackingNumber    string         `json:"tracking_number" bson:"tracking_number"`
	ClientID          string         `json:"client_id" bson:"client_id"`
	Sender            Person         `json:"sender" bson:"sender"`
	Origin            Address        `json:"origin" bson:"origin"`
	Destination       Address        `json:"destination" bson:"destination"`
	Package           Package        `json:"package" bson:"package"`
	ServiceType       string         `json:"service_type" bson:"service_type"`
	Status            ShipmentStatus `json:"status" bson:"status"`
	CreatedAt         time.Time      `json:"created_at" bson:"created_at"`
	EstimatedDelivery time.Time      `json:"estimated_delivery" bson:"estimated_delivery"`
	IdempotencyKey    string         `json:"idempotency_key,omitempty" bson:"idempotency_key,omitempty"`
	StatusHistory     []StatusHistoryEntry `json:"status_history" bson:"status_history"`
}
