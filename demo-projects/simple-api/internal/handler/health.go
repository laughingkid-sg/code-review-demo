package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/demo/simple-api/internal/store"
)

// HealthResponse represents the health check response payload.
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Database  string `json:"database"`
}

// HealthHandler handles health check requests.
type HealthHandler struct {
	store store.ProductStore
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(store store.ProductStore) *HealthHandler {
	return &HealthHandler{store: store}
}

// ServeHTTP implements the http.Handler interface for health checks.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	dbStatus := "connected"
	if h.store != nil {
		if err := h.store.Ping(r.Context()); err != nil {
			dbStatus = "disconnected"
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(HealthResponse{
				Status:    "unhealthy",
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Database:  dbStatus,
			})
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Database:  dbStatus,
	})
}
