package ports

import (
	"context"
	"time"
)

// LocationInput carries optional geographic coordinates for a tracking event.
type LocationInput struct {
	Lat float64
	Lng float64
}

// TrackingEventInput is the DTO passed from the transport layer to EventService.
type TrackingEventInput struct {
	TrackingNumber string
	Status         string
	Timestamp      time.Time
	Source         string
	Location       *LocationInput // optional
}

// EventService processes incoming tracking events.
type EventService interface {
	Process(ctx context.Context, event TrackingEventInput) error
}
