package handler

import "time"

type locationRequest struct {
	Lat float64 `json:"lat" validate:"required"`
	Lng float64 `json:"lng" validate:"required"`
}

type trackingEventRequest struct {
	TrackingNumber string           `json:"tracking_number" validate:"required"`
	Status         string           `json:"status"          validate:"required,oneof=picked_up in_warehouse in_transit delivered cancelled"`
	Timestamp      time.Time        `json:"timestamp"       validate:"required"`
	Source         string           `json:"source"          validate:"required"`
	Location       *locationRequest `json:"location"`
}

type acceptedResponse struct {
	Message string `json:"message"`
	Count   int    `json:"count,omitempty"`
}
