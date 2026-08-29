package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/demo/simple-api/internal/model"
	"github.com/demo/simple-api/internal/store"
)

// ProductHandler provides HTTP endpoints for product resource management.
type ProductHandler struct {
	store store.ProductStore
}

// NewProductHandler initializes a ProductHandler with the required storage layer.
func NewProductHandler(store store.ProductStore) *ProductHandler {
	return &ProductHandler{store: store}
}

// List handles GET /api/products with pagination, category filtering, and search.
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	page, _ := strconv.Atoi(query.Get("page"))
	limit, _ := strconv.Atoi(query.Get("limit"))

	filter := model.ProductListFilter{
		Category: query.Get("category"),
		Query:    query.Get("q"),
		Page:     page,
		Limit:    limit,
	}

	result, err := h.store.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch products")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// Get handles GET /api/products/{id}.
func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	product, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch product")
		return
	}

	writeJSON(w, http.StatusOK, product)
}

// Create handles POST /api/products.
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	product := model.Product{
		SKU:         req.SKU,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		Category:    req.Category,
	}

	if err := h.store.Create(r.Context(), &product); err != nil {
		if errors.Is(err, store.ErrDuplicateSKU) {
			writeError(w, http.StatusConflict, "product with this SKU already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create product")
		return
	}

	w.Header().Set("Location", fmt.Sprintf("/api/products/%d", product.ID))
	writeJSON(w, http.StatusCreated, product)
}

// Update handles PUT /api/products/{id}.
func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	var req model.UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	product := model.Product{
		ID:          id,
		SKU:         req.SKU,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		Category:    req.Category,
	}

	if err := h.store.Update(r.Context(), &product); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		if errors.Is(err, store.ErrDuplicateSKU) {
			writeError(w, http.StatusConflict, "product with this SKU already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update product")
		return
	}

	// Fetch fresh copy to ensure consistent timestamps
	updated, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusOK, product)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// Delete handles DELETE /api/products/{id}.
func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete product")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
