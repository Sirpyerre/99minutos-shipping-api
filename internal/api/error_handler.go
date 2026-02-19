package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"github.com/99minutos/shipping-system/internal/core/domain"
)

// errorResponse is the canonical error envelope for all API errors.
type errorResponse struct {
	Error string `json:"error"`
}

// NewHTTPErrorHandler returns an echo.HTTPErrorHandler that:
//   - Maps known domain errors to their appropriate HTTP status codes.
//   - Logs unexpected errors internally without leaking details to the client.
//   - Renders a consistent JSON envelope: {"error": "<message>"}.
func NewHTTPErrorHandler(log zerolog.Logger) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}

		code, msg := resolveError(err, log, c)
		_ = c.JSON(code, errorResponse{Error: msg})
	}
}

func resolveError(err error, log zerolog.Logger, c echo.Context) (int, string) {
	// Echo's own errors (bind failures, 404 from router, etc.)
	var he *echo.HTTPError
	if errors.As(err, &he) {
		return he.Code, fmt.Sprintf("%v", he.Message)
	}

	// Known domain errors → deterministic HTTP codes.
	switch {
	case errors.Is(err, domain.ErrShipmentNotFound):
		return http.StatusNotFound, "shipment not found"
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden, "access forbidden"
	case errors.Is(err, domain.ErrInvalidTransition):
		return http.StatusUnprocessableEntity, err.Error()
	case errors.Is(err, domain.ErrInvalidCredentials):
		return http.StatusUnauthorized, "invalid credentials"
	case errors.Is(err, domain.ErrUserNotFound):
		return http.StatusNotFound, "user not found"
	case errors.Is(err, domain.ErrUserExists):
		return http.StatusConflict, "user already exists"
	}

	// Unexpected error: log the real cause, return a generic message.
	log.Error().
		Err(err).
		Str("method", c.Request().Method).
		Str("path", c.Path()).
		Msg("unhandled error")

	return http.StatusInternalServerError, "internal server error"
}
