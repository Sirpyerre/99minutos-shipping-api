package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/99minutos/shipping-system/internal/core/ports"
)

// ShipmentHandler handles HTTP requests for shipment operations.
type ShipmentHandler struct {
	service ports.ShipmentService
}

func NewShipmentHandler(service ports.ShipmentService) *ShipmentHandler {
	return &ShipmentHandler{service: service}
}

// List handles GET /v1/shipments.
//
// @Summary      List shipments
// @Tags         shipments
// @Produce      json
// @Security     BearerAuth
// @Param        status        query     string  false  "Filter by status (created, picked_up, in_warehouse, in_transit, delivered, cancelled)"
// @Param        service_type  query     string  false  "Filter by service type (same_day, next_day, standard)"
// @Param        search        query     string  false  "Partial match on tracking_number or sender name"
// @Param        date_from     query     string  false  "Created at >= date (YYYY-MM-DD)"
// @Param        date_to       query     string  false  "Created at <= date (YYYY-MM-DD)"
// @Param        page          query     int     false  "Page number (default 1)"
// @Param        limit         query     int     false  "Items per page (default 20, max 100)"
// @Success      200           {object}  listShipmentsResponse
// @Failure      400           {object}  errorResponse
// @Failure      401           {object}  errorResponse
// @Failure      500           {object}  errorResponse
// @Router       /v1/shipments [get]
func (h *ShipmentHandler) List(c echo.Context) error {
	role, clientID, err := ctxClaims(c)
	if err != nil {
		return err
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))

	dateFrom, err := parseDate(c.QueryParam("date_from"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "date_from must be YYYY-MM-DD")
	}
	dateTo, err := parseDate(c.QueryParam("date_to"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "date_to must be YYYY-MM-DD")
	}

	result, err := h.service.ListShipments(c.Request().Context(), ports.ListShipmentsInput{
		Role:        role,
		ClientID:    clientID,
		Status:      c.QueryParam("status"),
		ServiceType: c.QueryParam("service_type"),
		Search:      c.QueryParam("search"),
		DateFrom:    dateFrom,
		DateTo:      dateTo,
		Page:        page,
		Limit:       limit,
	})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, toListResponse(result))
}

// parseDate parses an optional YYYY-MM-DD query param into time.Time (zero if empty).
func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse("2006-01-02", s)
}
//
// @Summary      Get a shipment by tracking number
// @Tags         shipments
// @Produce      json
// @Security     BearerAuth
// @Param        tracking_number  path      string  true  "Tracking number (e.g. 99M-7A8B9C2D)"
// @Success      200              {object}  getShipmentResponse
// @Failure      403              {object}  errorResponse
// @Failure      404              {object}  errorResponse
// @Failure      500              {object}  errorResponse
// @Router       /v1/shipments/{tracking_number} [get]
func (h *ShipmentHandler) Get(c echo.Context) error {
	role, clientID, err := ctxClaims(c)
	if err != nil {
		return err
	}

	detail, err := h.service.GetShipment(c.Request().Context(), ports.GetShipmentInput{
		TrackingNumber: c.Param("tracking_number"),
		Role:           role,
		ClientID:       clientID,
	})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, toGetResponse(detail))
}

// Create handles POST /v1/shipments.
//
// @Summary      Create a new shipment
// @Tags         shipments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Idempotency-Key  header    string                 false  "Idempotency key to prevent duplicate submissions"
// @Param        body             body      createShipmentRequest  true   "Shipment details"
// @Success      201              {object}  createShipmentResponse
// @Failure      400              {object}  errorResponse
// @Failure      401              {object}  errorResponse
// @Failure      422              {object}  errorResponse
// @Failure      500              {object}  errorResponse
// @Router       /v1/shipments [post]
func (h *ShipmentHandler) Create(c echo.Context) error {
	var req createShipmentRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid payload")
	}
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
	}

	_, clientID, err := ctxClaims(c)
	if err != nil {
		return err
	}

	result, err := h.service.CreateShipment(
		c.Request().Context(),
		toCreateInput(req, clientID, c.Request().Header.Get("Idempotency-Key")),
	)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, toCreateResponse(result))
}
