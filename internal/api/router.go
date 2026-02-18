package api

import (
	"time"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/99minutos/shipping-system/internal/api/handler"
	"github.com/99minutos/shipping-system/internal/api/middleware"
	"github.com/99minutos/shipping-system/internal/core/service"
	mongoauth "github.com/99minutos/shipping-system/internal/infrastructure/db/mongo"
)

// NewRouter builds and returns the Echo instance with all routes registered.
func NewRouter(db *mongo.Database, rdb *redis.Client, jwtSecret string) *echo.Echo {
	e := echo.New()
	e.HideBanner = true

	// --- Global middleware ---
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.RequestID())
	e.Use(echomiddleware.Logger())

	// --- Dependencies ---
	authRepo := mongoauth.NewAuthRepository(db)
	authService := service.NewAuthService(authRepo, jwtSecret, 24*time.Hour)
	authHandler := handler.NewAuthHandler(authService)
	authMiddleware := middleware.Auth(jwtSecret)
	_ = authMiddleware

	// --- Auth routes ---
	e.POST("/auth/register", authHandler.Register)
	e.POST("/auth/login", authHandler.Login)

	// --- Health probes (no auth required) ---
	healthHandler := handler.NewHealthHandler()
	healthDepsHandler := handler.NewHealthDependenciesHandler(db, rdb)

	e.GET("/health", healthHandler.Liveness)            // liveness  – is the process alive?
	e.GET("/health/ready", healthDepsHandler.Readiness) // readiness – are dependencies up?

	return e
}
