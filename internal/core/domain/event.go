package domain

import "time"

// TrackingEvent represents a status update received from an external source.
type TrackingEvent struct {
	TrackingNumber string
	Status         ShipmentStatus
	Timestamp      time.Time
	Source         string
	Location       *Coordinates // optional
}
