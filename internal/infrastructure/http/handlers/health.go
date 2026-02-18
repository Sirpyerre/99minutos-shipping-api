package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// HealthHandler handles GET /health — liveness probe.
// Returns 200 immediately; confirms the process is alive.
type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Liveness(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// HealthDependenciesHandler handles GET /health/ready — readiness probe.
// Checks MongoDB and Redis connectivity before declaring the service ready.
type HealthDependenciesHandler struct {
	mongo *mongo.Database
	redis *redis.Client
}

func NewHealthDependenciesHandler(db *mongo.Database, rdb *redis.Client) *HealthDependenciesHandler {
	return &HealthDependenciesHandler{
		mongo: db,
		redis: rdb,
	}
}

type dependencyStatus struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type readinessResponse struct {
	Status       string                      `json:"status"`
	Dependencies map[string]dependencyStatus `json:"dependencies"`
}

func (h *HealthDependenciesHandler) Readiness(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 3*time.Second)
	defer cancel()

	deps := make(map[string]dependencyStatus)
	healthy := true

	// --- MongoDB ping ---
	if err := h.mongo.Client().Ping(ctx, nil); err != nil {
		deps["mongodb"] = dependencyStatus{Status: "unhealthy", Error: err.Error()}
		healthy = false
	} else {
		deps["mongodb"] = dependencyStatus{Status: "ok"}
	}

	// --- MongoDB collections reachable ---
	if healthy {
		if err := h.mongo.RunCommand(ctx, bson.D{{Key: "ping", Value: 1}}).Err(); err != nil {
			deps["mongodb"] = dependencyStatus{Status: "unhealthy", Error: err.Error()}
			healthy = false
		}
	}

	// --- Redis ping ---
	if _, err := h.redis.Ping(ctx).Result(); err != nil {
		deps["redis"] = dependencyStatus{Status: "unhealthy", Error: err.Error()}
		healthy = false
	} else {
		deps["redis"] = dependencyStatus{Status: "ok"}
	}

	status := "ok"
	httpStatus := http.StatusOK
	if !healthy {
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	return c.JSON(httpStatus, readinessResponse{
		Status:       status,
		Dependencies: deps,
	})
}
