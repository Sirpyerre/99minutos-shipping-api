package service

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/99minutos/shipping-system/internal/core/domain"
)

type stubAuthRepo struct {
	users map[string]*domain.User
}

func newStubAuthRepo() *stubAuthRepo {
	return &stubAuthRepo{users: make(map[string]*domain.User)}
}

func cloneUser(u *domain.User) *domain.User {
	if u == nil {
		return nil
	}
	clone := *u
	return &clone
}

func (r *stubAuthRepo) Create(_ context.Context, user *domain.User) (*domain.User, error) {
	if _, exists := r.users[user.Username]; exists {
		return nil, domain.ErrUserExists
	}
	copy := cloneUser(user)
	if copy.ID == "" {
		copy.ID = user.Username
	}
	r.users[copy.Username] = cloneUser(copy)
	return cloneUser(copy), nil
}

func (r *stubAuthRepo) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, u := range r.users {
		if u.Email == email {
			return cloneUser(u), nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func TestAuthService_Register_Success(t *testing.T) {
	repo := newStubAuthRepo()
	svc := NewAuthService(repo, "secret", time.Hour)

	user, err := svc.Register(context.Background(), "alice", "pass123", "alice@example.com", domain.RoleClient, "client_1")
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if user == nil {
		t.Fatalf("expected user, got nil")
	}
	if user.PasswordHash == "pass123" {
		t.Fatalf("expected password to be hashed")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("pass123")); err != nil {
		t.Fatalf("stored hash does not match password: %v", err)
	}
	if user.Role != domain.RoleClient {
		t.Fatalf("unexpected role: %s", user.Role)
	}
}

func TestAuthService_Register_Validation(t *testing.T) {
	repo := newStubAuthRepo()
	svc := NewAuthService(repo, "secret", time.Hour)

	if _, err := svc.Register(context.Background(), "", "pass", "", domain.RoleClient, ""); err != domain.ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}

	if _, err := svc.Register(context.Background(), "bob", "pass", "bob@example.com", "wrong", ""); err != domain.ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials for bad role, got %v", err)
	}
}

func TestAuthService_Register_Duplicate(t *testing.T) {
	repo := newStubAuthRepo()
	svc := NewAuthService(repo, "secret", time.Hour)

	_, _ = svc.Register(context.Background(), "bob", "pass", "bob@example.com", domain.RoleClient, "")
	if _, err := svc.Register(context.Background(), "bob", "pass2", "bob@example.com", domain.RoleClient, ""); err != domain.ErrUserExists {
		t.Fatalf("expected ErrUserExists, got %v", err)
	}
}

func TestAuthService_Login_Success(t *testing.T) {
	repo := newStubAuthRepo()
	svc := NewAuthService(repo, "secret", time.Hour)

	if _, err := svc.Register(context.Background(), "carol", "s3cret", "carol@example.com", domain.RoleAdmin, ""); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	token, user, err := svc.Login(context.Background(), "carol@example.com", "s3cret")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if token == "" {
		t.Fatalf("expected token, got empty")
	}
	if user == nil || user.Username != "carol" {
		t.Fatalf("unexpected user: %+v", user)
	}

	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte("secret"), nil
	})
	if err != nil || !parsed.Valid {
		t.Fatalf("token invalid: %v", err)
	}
	if claims["role"] != domain.RoleAdmin {
		t.Fatalf("expected role %s, got %v", domain.RoleAdmin, claims["role"])
	}
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	repo := newStubAuthRepo()
	svc := NewAuthService(repo, "secret", time.Hour)

	_, _ = svc.Register(context.Background(), "dave", "goodpass", "dave@example.com", domain.RoleClient, "")
	if _, _, err := svc.Login(context.Background(), "dave@example.com", "badpass"); err != domain.ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	repo := newStubAuthRepo()
	svc := NewAuthService(repo, "secret", time.Hour)

	if _, _, err := svc.Login(context.Background(), "ghost@example.com", "pass"); err != domain.ErrUserNotFound {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}
