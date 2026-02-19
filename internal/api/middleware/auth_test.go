package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func TestAuthMiddleware_ValidToken(t *testing.T) {
	e := echo.New()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username":  "alice",
		"role":      "admin",
		"client_id": "client_1",
	})
	signed, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	mw := Auth("secret")
	handler := mw(func(c echo.Context) error {
		called = true
		if c.Get("username") != "alice" {
			t.Fatalf("username not set")
		}
		if c.Get("role") != "admin" {
			t.Fatalf("role not set")
		}
		if c.Get("client_id") != "client_1" {
			t.Fatalf("client_id not set")
		}
		return c.NoContent(http.StatusOK)
	})

	if err := handler(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !called {
		t.Fatalf("next not called")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := Auth("secret")
	handler := mw(func(c echo.Context) error {
		t.Fatalf("should not reach next")
		return nil
	})

	if err := handler(c); err != nil {
		e.HTTPErrorHandler(err, c)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_InvalidHeaderFormat(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Token abc")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := Auth("secret")
	handler := mw(func(c echo.Context) error {
		t.Fatalf("should not reach next")
		return nil
	})

	if err := handler(c); err != nil {
		e.HTTPErrorHandler(err, c)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer not-a-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := Auth("secret")
	handler := mw(func(c echo.Context) error {
		t.Fatalf("should not reach next")
		return nil
	})

	if err := handler(c); err != nil {
		e.HTTPErrorHandler(err, c)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}
