package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/99minutos/shipping-system/internal/core/domain"
	"github.com/99minutos/shipping-system/internal/core/ports"
)

type AuthHandler struct {
	authService ports.AuthService
}

func NewAuthHandler(authService ports.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email,omitempty"`
	Role     string `json:"role"`
	ClientID string `json:"client_id"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string       `json:"token,omitempty"`
	User  *domain.User `json:"user,omitempty"`
}

// Register creates a user and returns the created user (no token issued here).
func (h *AuthHandler) Register(c echo.Context) error {
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}

	user, err := h.authService.Register(c.Request().Context(), req.Username, req.Password, req.Email, req.Role, req.ClientID)
	if err != nil {
		status := http.StatusInternalServerError
		switch err {
		case domain.ErrUserExists:
			status = http.StatusConflict
		case domain.ErrInvalidCredentials:
			status = http.StatusBadRequest
		}
		return c.JSON(status, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, authResponse{User: user})
}

// Login authenticates a user and returns a JWT plus user info.
func (h *AuthHandler) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}

	token, user, err := h.authService.Login(c.Request().Context(), req.Username, req.Password)
	if err != nil {
		status := http.StatusUnauthorized
		switch err {
		case domain.ErrInvalidCredentials:
			status = http.StatusUnauthorized
		case domain.ErrUserNotFound:
			status = http.StatusNotFound
		}
		return c.JSON(status, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, authResponse{Token: token, User: user})
}
