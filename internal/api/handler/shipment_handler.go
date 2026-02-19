package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/99minutos/shipping-system/internal/core/domain"
	"github.com/99minutos/shipping-system/internal/core/ports"
)

// ShipmentHandler handles HTTP requests for shipment operations.
type ShipmentHandler struct {
	service ports.ShipmentService
}

func NewShipmentHandler(service ports.ShipmentService) *ShipmentHandler {
	return &ShipmentHandler{service: service}
}

// --- Request / Response types ---

type coordinatesRequest struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type addressRequest struct {
	Address     string             `json:"address"`
	City        string             `json:"city"`
	ZipCode     string             `json:"zip_code"`
	Coordinates coordinatesRequest `json:"coordinates"`
}

type senderRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

type dimensionsRequest struct {
	LengthCm float64 `json:"length_cm"`
	WidthCm  float64 `json:"width_cm"`
	HeightCm float64 `json:"height_cm"`
}

type packageRequest struct {
	WeightKg      float64           `json:"weight_kg"`
	Dimensions    dimensionsRequest `json:"dimensions"`
	Description   string            `json:"description"`
	DeclaredValue float64           `json:"declared_value"`
	Currency      string            `json:"currency"`
}

type createShipmentRequest struct {
	Sender      senderRequest  `json:"sender"`
	Origin      addressRequest `json:"origin"`
	Destination addressRequest `json:"destination"`
	Package     packageRequest `json:"package"`
	ServiceType string         `json:"service_type"`
}

type shipmentLinks struct {
	Self   string `json:"self"`
	Events string `json:"events"`
}

type createShipmentResponse struct {
	TrackingNumber    string        `json:"tracking_number"`
	Status            string        `json:"status"`
	CreatedAt         string        `json:"created_at"`
	EstimatedDelivery string        `json:"estimated_delivery"`
	Links             shipmentLinks `json:"_links"`
}

type getShipmentResponse struct {
	TrackingNumber    string                    `json:"tracking_number"`
	Status            string                    `json:"status"`
	ServiceType       string                    `json:"service_type"`
	CreatedAt         string                    `json:"created_at"`
	EstimatedDelivery string                    `json:"estimated_delivery"`
	Sender            ports.SenderInput         `json:"sender"`
	Origin            ports.AddressInput        `json:"origin"`
	Destination       ports.AddressInput        `json:"destination"`
	Package           ports.PackageInput        `json:"package"`
	StatusHistory     []ports.StatusHistoryItem `json:"status_history"`
	Links             shipmentLinks             `json:"_links"`
}

// Get handles GET /v1/shipments/:tracking_number.
//
// @Summary      Get a shipment by tracking number
// @Tags         shipments
// @Produce      json
// @Security     BearerAuth
// @Param        tracking_number  path      string  true  "Tracking number (e.g. 99M-7A8B9C2D)"
// @Success      200              {object}  getShipmentResponse
// @Failure      403              {object}  map[string]string
// @Failure      404              {object}  map[string]string
// @Failure      500              {object}  map[string]string
// @Router       /v1/shipments/{tracking_number} [get]
func (h *ShipmentHandler) Get(c echo.Context) error {
	trackingNumber := c.Param("tracking_number")
	role, _ := c.Get("role").(string)
	clientID, _ := c.Get("client_id").(string)

	detail, err := h.service.GetShipment(c.Request().Context(), ports.GetShipmentInput{
		TrackingNumber: trackingNumber,
		Role:           role,
		ClientID:       clientID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrShipmentNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "shipment not found"})
		}
		if errors.Is(err, domain.ErrForbidden) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "access forbidden"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	return c.JSON(http.StatusOK, getShipmentResponse{
		TrackingNumber:    detail.TrackingNumber,
		Status:            detail.Status,
		ServiceType:       detail.ServiceType,
		CreatedAt:         detail.CreatedAt,
		EstimatedDelivery: detail.EstimatedDelivery,
		Sender:            detail.Sender,
		Origin:            detail.Origin,
		Destination:       detail.Destination,
		Package:           detail.Package,
		StatusHistory:     detail.StatusHistory,
		Links: shipmentLinks{
			Self:   "/shipments/" + detail.TrackingNumber,
			Events: "/events/" + detail.TrackingNumber,
		},
	})
}
//
// @Summary      Create a new shipment
// @Tags         shipments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Idempotency-Key  header    string                 false  "Idempotency key to prevent duplicate submissions"
// @Param        body             body      createShipmentRequest  true   "Shipment details"
// @Success      201              {object}  createShipmentResponse
// @Failure      400              {object}  map[string]string
// @Failure      401              {object}  map[string]string
// @Failure      500              {object}  map[string]string
// @Router       /v1/shipments [post]
func (h *ShipmentHandler) Create(c echo.Context) error {
	var req createShipmentRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}

	clientID, _ := c.Get("client_id").(string)
	idempotencyKey := c.Request().Header.Get("Idempotency-Key")

	result, err := h.service.CreateShipment(c.Request().Context(), ports.CreateShipmentInput{
		Sender: ports.SenderInput{
			Name:  req.Sender.Name,
			Email: req.Sender.Email,
			Phone: req.Sender.Phone,
		},
		Origin: ports.AddressInput{
			Address: req.Origin.Address,
			City:    req.Origin.City,
			ZipCode: req.Origin.ZipCode,
			Coordinates: ports.CoordinatesInput{
				Lat: req.Origin.Coordinates.Lat,
				Lng: req.Origin.Coordinates.Lng,
			},
		},
		Destination: ports.AddressInput{
			Address: req.Destination.Address,
			City:    req.Destination.City,
			ZipCode: req.Destination.ZipCode,
			Coordinates: ports.CoordinatesInput{
				Lat: req.Destination.Coordinates.Lat,
				Lng: req.Destination.Coordinates.Lng,
			},
		},
		Package: ports.PackageInput{
			WeightKg: req.Package.WeightKg,
			Dimensions: ports.DimensionsInput{
				LengthCm: req.Package.Dimensions.LengthCm,
				WidthCm:  req.Package.Dimensions.WidthCm,
				HeightCm: req.Package.Dimensions.HeightCm,
			},
			Description:   req.Package.Description,
			DeclaredValue: req.Package.DeclaredValue,
			Currency:      req.Package.Currency,
		},
		ServiceType:    req.ServiceType,
		ClientID:       clientID,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create shipment"})
	}

	resp := createShipmentResponse{
		TrackingNumber:    result.TrackingNumber,
		Status:            result.Status,
		CreatedAt:         result.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		EstimatedDelivery: result.EstimatedDelivery.UTC().Format("2006-01-02T15:04:05Z"),
		Links: shipmentLinks{
			Self:   "/shipments/" + result.TrackingNumber,
			Events: "/events/" + result.TrackingNumber,
		},
	}

	return c.JSON(http.StatusCreated, resp)
}