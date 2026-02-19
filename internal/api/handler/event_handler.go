package handler

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/99minutos/shipping-system/internal/core/ports"
)

// EventDispatcher is the interface the handler uses to enqueue events.
type EventDispatcher interface {
	Enqueue(event ports.TrackingEventInput)
	EnqueueBatch(events []ports.TrackingEventInput)
}

// EventHandler handles tracking event ingestion.
type EventHandler struct {
	dispatcher EventDispatcher
}

// NewEventHandler creates an EventHandler backed by the given dispatcher.
func NewEventHandler(dispatcher EventDispatcher) *EventHandler {
	return &EventHandler{dispatcher: dispatcher}
}

// Receive handles POST /v1/events — enqueues a single event, returns 202.
//
// @Summary      Ingest a single tracking event
// @Tags         events
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      trackingEventRequest  true  "Tracking event"
// @Success      202   {object}  acceptedResponse
// @Failure      400   {object}  errorResponse
// @Failure      401   {object}  errorResponse
// @Failure      422   {object}  errorResponse
// @Router       /v1/events [post]
func (h *EventHandler) Receive(c echo.Context) error {
	var req trackingEventRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid payload")
	}
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
	}

	h.dispatcher.Enqueue(toEventInput(req))
	return c.JSON(http.StatusAccepted, acceptedResponse{Message: "event accepted"})
}

// ReceiveBatch handles POST /v1/events/batch — enqueues a batch of events, returns 202.
//
// @Summary      Ingest a batch of tracking events
// @Tags         events
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      []trackingEventRequest  true  "Array of tracking events"
// @Success      202   {object}  acceptedResponse
// @Failure      400   {object}  errorResponse
// @Failure      401   {object}  errorResponse
// @Failure      422   {object}  errorResponse
// @Router       /v1/events/batch [post]
func (h *EventHandler) ReceiveBatch(c echo.Context) error {
	var reqs []trackingEventRequest
	if err := c.Bind(&reqs); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid payload")
	}
	if len(reqs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "batch cannot be empty")
	}

	inputs := make([]ports.TrackingEventInput, 0, len(reqs))
	for i, req := range reqs {
		if err := c.Validate(&req); err != nil {
			return echo.NewHTTPError(http.StatusUnprocessableEntity,
				fmt.Sprintf("event[%d]: %s", i, err.Error()))
		}
		inputs = append(inputs, toEventInput(req))
	}

	h.dispatcher.EnqueueBatch(inputs)
	return c.JSON(http.StatusAccepted, acceptedResponse{
		Message: "events accepted",
		Count:   len(inputs),
	})
}

// toEventInput maps the HTTP request to the service DTO.
func toEventInput(r trackingEventRequest) ports.TrackingEventInput {
	in := ports.TrackingEventInput{
		TrackingNumber: r.TrackingNumber,
		Status:         r.Status,
		Timestamp:      r.Timestamp,
		Source:         r.Source,
	}
	if r.Location != nil {
		in.Location = &ports.LocationInput{Lat: r.Location.Lat, Lng: r.Location.Lng}
	}
	return in
}