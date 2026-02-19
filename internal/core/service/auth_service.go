package service

import (
	"context"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/99minutos/shipping-system/internal/core/domain"
	"github.com/99minutos/shipping-system/internal/core/ports"
)

// AuthService implements registration and login.
type AuthService struct {
	repo      ports.AuthRepository
	jwtSecret string
	tokenTTL  time.Duration
}

func NewAuthService(repo ports.AuthRepository, jwtSecret string, tokenTTL time.Duration) *AuthService {
	if tokenTTL <= 0 {
		tokenTTL = 24 * time.Hour
	}
	return &AuthService{repo: repo, jwtSecret: jwtSecret, tokenTTL: tokenTTL}
}

func (s *AuthService) Register(ctx context.Context, username, password, email, role, clientID string) (*domain.User, error) {
	if username == "" || password == "" || role == "" || email == "" {
		return nil, domain.ErrInvalidCredentials
	}
	if role != domain.RoleAdmin && role != domain.RoleClient {
		return nil, domain.ErrInvalidCredentials
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	user := &domain.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		Role:         role,
		ClientID:     clientID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	created, err := s.repo.Create(ctx, user)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, *domain.User, error) {
	if email == "" || password == "" {
		return "", nil, domain.ErrInvalidCredentials
	}

	log.Printf("pass %s", password)
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		log.Printf("Login failed for email %s: %v", email, err)
		return "", nil, err
	}

	log.Printf("User found for email %s:, password_hash=%s", email, user.PasswordHash)
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		log.Printf("Invalid password for email %s", email)
		return "", nil, domain.ErrInvalidCredentials
	}

	token, err := s.generateToken(user)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil
}

func (s *AuthService) generateToken(user *domain.User) (string, error) {
	claims := jwt.MapClaims{
		"username":  user.Username,
		"role":      user.Role,
		"client_id": user.ClientID,
		"exp":       time.Now().Add(s.tokenTTL).Unix(),
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(s.jwtSecret))
}
