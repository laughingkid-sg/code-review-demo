package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/demo/simple-api/internal/middleware"
	"github.com/demo/simple-api/internal/model"
	"github.com/demo/simple-api/internal/store"
)

func setupTestRouter(t *testing.T) (http.Handler, store.ProductStore) {
	t.Helper()
	s, err := store.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create memory store: %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})

	apiKey := "test-secret-key"
	authMiddleware := middleware.NewAPIKeyMiddleware(apiKey)

	healthH := NewHealthHandler(s)
	productH := NewProductHandler(s)

	mux := http.NewServeMux()
	mux.Handle("GET /api/health", healthH)
	mux.HandleFunc("GET /api/products", productH.List)
	mux.HandleFunc("GET /api/products/{id}", productH.Get)
	mux.HandleFunc("POST /api/products", authMiddleware.RequireKeyFunc(productH.Create))
	mux.HandleFunc("PUT /api/products/{id}", authMiddleware.RequireKeyFunc(productH.Update))
	mux.HandleFunc("DELETE /api/products/{id}", authMiddleware.RequireKeyFunc(productH.Delete))

	return mux, s
}

func TestHealthHandler(t *testing.T) {
	router, _ := setupTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if resp.Status != "ok" || resp.Database != "connected" {
		t.Errorf("unexpected health response: %+v", resp)
	}
}

func TestProductHandler_Create_And_Get(t *testing.T) {
	router, _ := setupTestRouter(t)

	t.Run("Unauthorized when missing API key", func(t *testing.T) {
		payload := `{"sku":"KEY-1","name":"Test","price":10,"stock":5,"category":"General"}`
		req := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewBufferString(payload))
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("Create success with valid key", func(t *testing.T) {
		payload := `{"sku":"KEY-101","name":"Wireless Mouse","description":"Quiet click mouse","price":29.99,"stock":50,"category":"Accessories"}`
		req := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewBufferString(payload))
		req.Header.Set("X-API-Key", "test-secret-key")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d (body: %s)", rec.Code, rec.Body.String())
		}

		var created model.Product
		if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if created.ID <= 0 || created.SKU != "KEY-101" {
			t.Errorf("unexpected created product data: %+v", created)
		}

		// Verify Get
		getReq := httptest.NewRequest(http.MethodGet, "/api/products/1", nil)
		getRec := httptest.NewRecorder()
		router.ServeHTTP(getRec, getReq)

		if getRec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK from get, got %d", getRec.Code)
		}
	})

	t.Run("Create validation error on invalid SKU", func(t *testing.T) {
		payload := `{"sku":"!@#","name":"Bad SKU","price":10,"stock":5,"category":"General"}`
		req := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewBufferString(payload))
		req.Header.Set("X-API-Key", "test-secret-key")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", rec.Code)
		}
	})

	t.Run("Create duplicate SKU conflict", func(t *testing.T) {
		payload := `{"sku":"KEY-101","name":"Duplicate SKU Product","price":40,"stock":1,"category":"Accessories"}`
		req := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewBufferString(payload))
		req.Header.Set("X-API-Key", "test-secret-key")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict, got %d", rec.Code)
		}
	})
}

func TestProductHandler_List_And_Filter(t *testing.T) {
	router, _ := setupTestRouter(t)

	// Seed 2 products
	products := []string{
		`{"sku":"LAP-01","name":"Pro Laptop","price":1200,"stock":10,"category":"Computers"}`,
		`{"sku":"MOU-01","name":"Ergo Mouse","price":45,"stock":25,"category":"Accessories"}`,
	}

	for _, p := range products {
		req := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewBufferString(p))
		req.Header.Set("X-API-Key", "test-secret-key")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("failed to seed product: %s", rec.Body.String())
		}
	}

	t.Run("List all", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/products", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}

		var res model.PaginatedProducts
		if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
			t.Fatalf("failed to decode list response: %v", err)
		}

		if res.Pagination.TotalItems != 2 || len(res.Data) != 2 {
			t.Errorf("unexpected list count: %+v", res.Pagination)
		}
	})

	t.Run("Filter by category Computers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/products?category=Computers", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		var res model.PaginatedProducts
		_ = json.NewDecoder(rec.Body).Decode(&res)

		if len(res.Data) != 1 || res.Data[0].SKU != "LAP-01" {
			t.Errorf("unexpected filtered results: %+v", res.Data)
		}
	})
}

func TestProductHandler_Update_And_Delete(t *testing.T) {
	router, _ := setupTestRouter(t)

	// Create a product
	createPayload := `{"sku":"TAB-01","name":"Tablet Pro","price":499.99,"stock":15,"category":"Electronics"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewBufferString(createPayload))
	createReq.Header.Set("X-API-Key", "test-secret-key")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("failed to create initial product: %d", createRec.Code)
	}

	t.Run("Update product success", func(t *testing.T) {
		updatePayload := `{"sku":"TAB-01-V2","name":"Tablet Pro Max","price":549.99,"stock":20,"category":"Electronics"}`
		req := httptest.NewRequest(http.MethodPut, "/api/products/1", bytes.NewBufferString(updatePayload))
		req.Header.Set("X-API-Key", "test-secret-key")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d (body: %s)", rec.Code, rec.Body.String())
		}

		var updated model.Product
		_ = json.NewDecoder(rec.Body).Decode(&updated)
		if updated.Name != "Tablet Pro Max" || updated.SKU != "TAB-01-V2" || updated.Price != 549.99 {
			t.Errorf("updated product mismatch: %+v", updated)
		}
	})

	t.Run("Update non-existent product returns 404", func(t *testing.T) {
		updatePayload := `{"sku":"TAB-GHOST","name":"Ghost","price":10,"stock":1,"category":"Electronics"}`
		req := httptest.NewRequest(http.MethodPut, "/api/products/9999", bytes.NewBufferString(updatePayload))
		req.Header.Set("X-API-Key", "test-secret-key")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found, got %d", rec.Code)
		}
	})

	t.Run("Delete product success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/products/1", nil)
		req.Header.Set("X-API-Key", "test-secret-key")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204 No Content, got %d", rec.Code)
		}

		// Verify deletion
		getReq := httptest.NewRequest(http.MethodGet, "/api/products/1", nil)
		getRec := httptest.NewRecorder()
		router.ServeHTTP(getRec, getReq)

		if getRec.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found after delete, got %d", getRec.Code)
		}
	})

	t.Run("Delete non-existent product returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/products/9999", nil)
		req.Header.Set("X-API-Key", "test-secret-key")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found, got %d", rec.Code)
		}
	})
}
