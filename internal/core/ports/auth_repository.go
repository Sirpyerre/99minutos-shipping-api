package ports

import (
	"context"

	"github.com/99minutos/shipping-system/internal/core/domain"
)

// AuthRepository defines the interface for user authentication persistence.
type AuthRepository interface {
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	Create(ctx context.Context, user *domain.User) (*domain.User, error)
}

