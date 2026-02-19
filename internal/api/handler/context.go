package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/99minutos/shipping-system/internal/core/domain"
)

// ctxClaims extracts the auth claims injected by the Auth middleware and
// performs a fast-fail check before any service call:
//   - role must be non-empty (presence proves the middleware ran).
//   - client role requires a non-empty client_id; without it the JWT is
//     structurally valid but operationally unusable — reject with 401.
func ctxClaims(c echo.Context) (role, clientID string, err error) {
	role, _ = c.Get("role").(string)
	if role == "" {
		return "", "", echo.NewHTTPError(http.StatusUnauthorized, "missing authentication claims")
	}

	clientID, _ = c.Get("client_id").(string)
	if role == domain.RoleClient && clientID == "" {
		return "", "", echo.NewHTTPError(http.StatusUnauthorized, "token missing client identity")
	}

	return role, clientID, nil
}
