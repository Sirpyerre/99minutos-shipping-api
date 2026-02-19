package handler

import "time"

// errorResponse is the standard error envelope returned on all 4xx/5xx responses.
type errorResponse struct {
	Error string `json:"error"`
}

// --- Request / Response types ---

type coordinatesRequest struct {
	Lat float64 `json:"lat" validate:"required"`
	Lng float64 `json:"lng" validate:"required"`
}

type addressRequest struct {
	Address     string             `json:"address"      validate:"required"`
	City        string             `json:"city"         validate:"required"`
	ZipCode     string             `json:"zip_code"     validate:"required"`
	Coordinates coordinatesRequest `json:"coordinates"  validate:"required"`
}

type senderRequest struct {
	Name  string `json:"name"  validate:"required"`
	Email string `json:"email" validate:"required,email"`
	Phone string `json:"phone" validate:"required"`
}

type dimensionsRequest struct {
	LengthCm float64 `json:"length_cm" validate:"required,gt=0"`
	WidthCm  float64 `json:"width_cm"  validate:"required,gt=0"`
	HeightCm float64 `json:"height_cm" validate:"required,gt=0"`
}

type packageRequest struct {
	WeightKg      float64           `json:"weight_kg"      validate:"required,gt=0"`
	Dimensions    dimensionsRequest `json:"dimensions"     validate:"required"`
	Description   string            `json:"description"    validate:"required"`
	DeclaredValue float64           `json:"declared_value" validate:"required,gt=0"`
	Currency      string            `json:"currency"       validate:"required"`
}

type createShipmentRequest struct {
	Sender      senderRequest  `json:"sender"       validate:"required"`
	Origin      addressRequest `json:"origin"       validate:"required"`
	Destination addressRequest `json:"destination"  validate:"required"`
	Package     packageRequest `json:"package"      validate:"required"`
	ServiceType string         `json:"service_type" validate:"required,oneof=same_day next_day standard"`
}

type shipmentLinks struct {
	Self   string `json:"self"`
	Events string `json:"events"`
}

type createShipmentResponse struct {
	TrackingNumber    string        `json:"tracking_number"`
	Status            string        `json:"status"`
	CreatedAt         time.Time     `json:"created_at"`
	EstimatedDelivery time.Time     `json:"estimated_delivery"`
	Links             shipmentLinks `json:"_links"`
}

// Response-only types owned by the transport layer.
// These are intentionally separate from ports/domain types so the JSON
// contract is not coupled to internal service changes.

type senderResponse struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

type coordinatesResponse struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type addressResponse struct {
	Address     string              `json:"address"`
	City        string              `json:"city"`
	ZipCode     string              `json:"zip_code"`
	Coordinates coordinatesResponse `json:"coordinates"`
}

type dimensionsResponse struct {
	LengthCm float64 `json:"length_cm"`
	WidthCm  float64 `json:"width_cm"`
	HeightCm float64 `json:"height_cm"`
}

type packageResponse struct {
	WeightKg      float64            `json:"weight_kg"`
	Dimensions    dimensionsResponse `json:"dimensions"`
	Description   string             `json:"description"`
	DeclaredValue float64            `json:"declared_value"`
	Currency      string             `json:"currency"`
}

type statusHistoryItemResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Notes     string    `json:"notes,omitempty"`
}

type getShipmentResponse struct {
	TrackingNumber    string                      `json:"tracking_number"`
	Status            string                      `json:"status"`
	ServiceType       string                      `json:"service_type"`
	CreatedAt         time.Time                   `json:"created_at"`
	EstimatedDelivery time.Time                   `json:"estimated_delivery"`
	Sender            senderResponse              `json:"sender"`
	Origin            addressResponse             `json:"origin"`
	Destination       addressResponse             `json:"destination"`
	Package           packageResponse             `json:"package"`
	StatusHistory     []statusHistoryItemResponse `json:"status_history"`
	Links             shipmentLinks               `json:"_links"`
}

// shipmentSummaryResponse is the lightweight item used in list responses.
// It intentionally omits status_history to keep payloads small.
type shipmentSummaryResponse struct {
	TrackingNumber    string          `json:"tracking_number"`
	Status            string          `json:"status"`
	ServiceType       string          `json:"service_type"`
	CreatedAt         time.Time       `json:"created_at"`
	EstimatedDelivery time.Time       `json:"estimated_delivery"`
	Sender            senderResponse  `json:"sender"`
	Origin            addressResponse `json:"origin"`
	Destination       addressResponse `json:"destination"`
	Links             shipmentLinks   `json:"_links"`
}

type paginationResponse struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

type listShipmentsResponse struct {
	Data       []shipmentSummaryResponse `json:"data"`
	Pagination paginationResponse        `json:"pagination"`
}
