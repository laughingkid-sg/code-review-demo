package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
)

// APIKeyMiddleware provides middleware for validating the X-API-Key request header.
type APIKeyMiddleware struct {
	expectedKey string
}

// NewAPIKeyMiddleware creates a new APIKeyMiddleware with the provided secret key.
func NewAPIKeyMiddleware(expectedKey string) *APIKeyMiddleware {
	return &APIKeyMiddleware{
		expectedKey: expectedKey,
	}
}

// RequireKey wraps an http.Handler to enforce API key presence and correctness.
func (m *APIKeyMiddleware) RequireKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" || subtle.ConstantTimeCompare([]byte(apiKey), []byte(m.expectedKey)) != 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "unauthorized: invalid or missing API key",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireKeyFunc wraps an http.HandlerFunc to enforce API key authentication.
func (m *APIKeyMiddleware) RequireKeyFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" || subtle.ConstantTimeCompare([]byte(apiKey), []byte(m.expectedKey)) != 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "unauthorized: invalid or missing API key",
			})
			return
		}
		next(w, r)
	}
}
