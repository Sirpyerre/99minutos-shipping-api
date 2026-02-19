package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/99minutos/shipping-system/internal/core/domain"
)

type stubAuthService struct {
	registerFn func(ctx context.Context, username, password, email, role, clientID string) (*domain.User, error)
	loginFn    func(ctx context.Context, email, password string) (string, *domain.User, error)
}

func (s *stubAuthService) Register(ctx context.Context, username, password, email, role, clientID string) (*domain.User, error) {
	return s.registerFn(ctx, username, password, email, role, clientID)
}

func (s *stubAuthService) Login(ctx context.Context, email, password string) (string, *domain.User, error) {
	return s.loginFn(ctx, email, password)
}

func TestAuthHandler_Register_Success(t *testing.T) {
	e := echo.New()
	stub := &stubAuthService{
		registerFn: func(ctx context.Context, username, password, email, role, clientID string) (*domain.User, error) {
			if username != "alice" || role != "client" || clientID != "client_1" {
				t.Fatalf("unexpected args: %s %s %s", username, role, clientID)
			}
			return &domain.User{Username: username, Role: role, ClientID: clientID}, nil
		},
	}
	handler := NewAuthHandler(stub)

	body := strings.NewReader(`{"username":"alice","password":"secret","email":"a@example.com","role":"client","client_id":"client_1"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.Register(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	user, ok := resp["user"].(map[string]any)
	if !ok {
		t.Fatalf("expected user in response")
	}
	if user["username"] != "alice" || user["role"] != "client" || user["client_id"] != "client_1" {
		t.Fatalf("unexpected user payload: %+v", user)
	}
}

func TestAuthHandler_Register_UserExists(t *testing.T) {
	e := echo.New()
	stub := &stubAuthService{
		registerFn: func(ctx context.Context, username, password, email, role, clientID string) (*domain.User, error) {
			return nil, domain.ErrUserExists
		},
	}
	handler := NewAuthHandler(stub)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{"username":"bob"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = handler.Register(c)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestAuthHandler_Register_InvalidPayload(t *testing.T) {
	e := echo.New()
	stub := &stubAuthService{
		registerFn: func(ctx context.Context, username, password, email, role, clientID string) (*domain.User, error) {
			t.Fatalf("should not be called")
			return nil, nil
		},
	}
	handler := NewAuthHandler(stub)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader("not-json"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = handler.Register(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAuthHandler_Login_Success(t *testing.T) {
	e := echo.New()
	stub := &stubAuthService{
		loginFn: func(ctx context.Context, email, password string) (string, *domain.User, error) {
			if email != "alice@example.com" || password != "secret" {
				t.Fatalf("unexpected args: %s %s", email, password)
			}
			return "token123", &domain.User{Username: "alice", Role: "admin", ClientID: ""}, nil
		},
	}
	handler := NewAuthHandler(stub)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"alice@example.com","password":"secret"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.Login(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if resp["token"] != "token123" {
		t.Fatalf("expected token, got %v", resp["token"])
	}
	user, ok := resp["user"].(map[string]any)
	if !ok || user["username"] != "alice" || user["role"] != "admin" {
		t.Fatalf("unexpected user payload: %+v", user)
	}
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	e := echo.New()
	stub := &stubAuthService{
		loginFn: func(ctx context.Context, email, password string) (string, *domain.User, error) {
			return "", nil, domain.ErrInvalidCredentials
		},
	}
	handler := NewAuthHandler(stub)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"alice@example.com","password":"bad"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = handler.Login(c)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthHandler_Login_UserNotFound(t *testing.T) {
	e := echo.New()
	stub := &stubAuthService{
		loginFn: func(ctx context.Context, email, password string) (string, *domain.User, error) {
			return "", nil, domain.ErrUserNotFound
		},
	}
	handler := NewAuthHandler(stub)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"ghost@example.com","password":"pwd"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = handler.Login(c)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestAuthHandler_Login_InvalidPayload(t *testing.T) {
	e := echo.New()
	stub := &stubAuthService{
		loginFn: func(ctx context.Context, email, password string) (string, *domain.User, error) {
			t.Fatalf("should not be called")
			return "", nil, nil
		},
	}
	handler := NewAuthHandler(stub)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader("{"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = handler.Login(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
