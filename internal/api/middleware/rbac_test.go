package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestRBAC_Allows(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("role", "admin")

	called := false
	mw := RBAC("admin", "client")
	handler := mw(func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})

	if err := handler(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !called {
		t.Fatalf("next handler not called")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRBAC_Forbids(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("role", "guest")

	mw := RBAC("admin", "client")
	handler := mw(func(c echo.Context) error {
		t.Fatalf("should not reach next handler")
		return nil
	})

	_ = handler(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}
