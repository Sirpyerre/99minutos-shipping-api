package ports

import (
	"context"

	"github.com/99minutos/shipping-system/internal/core/domain"
)

type AuthService interface {
	Register(ctx context.Context, username, password, email, role, clientID string) (*domain.User, error)
	Login(ctx context.Context, username, password string) (string, *domain.User, error)
}
