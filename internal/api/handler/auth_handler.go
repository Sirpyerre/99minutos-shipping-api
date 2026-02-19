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
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token     string `json:"token"`
	TokenType string `json:"token_type"`
	ExpiresIn int    `json:"expires_in"` // seconds
}

// Register creates a new user account.
//
// @Summary      Register a new user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      registerRequest  true  "User registration details"
// @Success      201   {object}  authResponse
// @Failure      400   {object}  map[string]string
// @Failure      409   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /auth/register [post]
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

	_ = user
	return c.JSON(http.StatusCreated, map[string]string{"message": "user created"})
}

// Login authenticates a user and returns a JWT token.
//
// @Summary      Login
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      loginRequest  true  "Login credentials"
// @Success      200   {object}  authResponse
// @Failure      400   {object}  map[string]string
// @Failure      401   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}

	token, _, err := h.authService.Login(c.Request().Context(), req.Email, req.Password)
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

	return c.JSON(http.StatusOK, authResponse{
		Token:     token,
		TokenType: "Bearer",
		ExpiresIn: 86400, // 24 h, matches service default TTL
	})
}
