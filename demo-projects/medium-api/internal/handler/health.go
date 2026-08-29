package handler

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/demo/medium-api/internal/cache"
	"github.com/gin-gonic/gin"
)

// HealthResponse represents the health endpoint response payload.
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Database  string `json:"database"`
	Cache     string `json:"cache"`
}

// HealthHandler provides health check endpoints.
type HealthHandler struct {
	db    *sql.DB
	cache cache.Cache
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(db *sql.DB, cache cache.Cache) *HealthHandler {
	return &HealthHandler{
		db:    db,
		cache: cache,
	}
}

// Check verifies database and Redis connectivity.
func (h *HealthHandler) Check(c *gin.Context) {
	dbStatus := "connected"
	cacheStatus := "connected"
	isHealthy := true

	if h.db != nil {
		if err := h.db.PingContext(c.Request.Context()); err != nil {
			dbStatus = "disconnected"
			isHealthy = false
		}
	}

	if h.cache != nil {
		if err := h.cache.Ping(c.Request.Context()); err != nil {
			cacheStatus = "disconnected"
			isHealthy = false
		}
	}

	statusCode := http.StatusOK
	statusStr := "ok"
	if !isHealthy {
		statusCode = http.StatusServiceUnavailable
		statusStr = "unhealthy"
	}

	c.JSON(statusCode, HealthResponse{
		Status:    statusStr,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Database:  dbStatus,
		Cache:     cacheStatus,
	})
}
