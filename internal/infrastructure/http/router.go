package http

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/99minutos/shipping-system/internal/infrastructure/http/handlers"
)

// NewRouter builds and returns the Echo instance with all routes registered.
func NewRouter(db *mongo.Database, rdb *redis.Client) *echo.Echo {
	e := echo.New()
	e.HideBanner = true

	// --- Global middleware ---
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.Logger())

	// --- Health probes (no auth required) ---
	healthHandler := handlers.NewHealthHandler()
	healthDepsHandler := handlers.NewHealthDependenciesHandler(db, rdb)

	e.GET("/health", healthHandler.Liveness)            // liveness  – is the process alive?
	e.GET("/health/ready", healthDepsHandler.Readiness)  // readiness – are dependencies up?

	return e
}
