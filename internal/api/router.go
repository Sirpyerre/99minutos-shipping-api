package api

import (
	"time"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"
	echoswagger "github.com/swaggo/echo-swagger"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/99minutos/shipping-system/internal/api/handler"
	"github.com/99minutos/shipping-system/internal/api/middleware"
	"github.com/99minutos/shipping-system/internal/core/service"
	mongoinfra "github.com/99minutos/shipping-system/internal/infrastructure/db/mongo"
	"github.com/99minutos/shipping-system/internal/pkg/logger"
)

// NewRouter builds and returns the Echo instance with all routes registered.
func NewRouter(db *mongo.Database, rdb *redis.Client, jwtSecret string) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.Validator = handler.NewValidator()

	// --- Global middleware ---
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.RequestID())
	e.Use(echomiddleware.Logger())

	// --- Dependencies ---
	log := logger.Init(logger.Options{Pretty: true})

	e.HTTPErrorHandler = NewHTTPErrorHandler(log)

	authRepo := mongoinfra.NewAuthRepository(db)
	authService := service.NewAuthService(authRepo, jwtSecret, 24*time.Hour)
	authHandler := handler.NewAuthHandler(authService)

	shipmentRepo := mongoinfra.NewShipmentRepository(db)
	shipmentService := service.NewShipmentService(shipmentRepo, log)
	shipmentHandler := handler.NewShipmentHandler(shipmentService)

	authMiddleware := middleware.Auth(jwtSecret)

	// --- Auth routes (public) ---
	e.POST("/auth/register", authHandler.Register)
	e.POST("/auth/login", authHandler.Login)

	// --- Health probes (no auth required) ---
	healthHandler := handler.NewHealthHandler()
	healthDepsHandler := handler.NewHealthDependenciesHandler(db, rdb)

	e.GET("/health", healthHandler.Liveness)
	e.GET("/health/ready", healthDepsHandler.Readiness)

	// --- Swagger UI ---
	e.GET("/swagger/*", echoswagger.WrapHandler)

	// --- v1 API (JWT protected) ---
	v1 := e.Group("/v1", authMiddleware)
	v1.GET("/shipments", shipmentHandler.List)
	v1.POST("/shipments", shipmentHandler.Create)
	v1.GET("/shipments/:tracking_number", shipmentHandler.Get)

	return e
}
