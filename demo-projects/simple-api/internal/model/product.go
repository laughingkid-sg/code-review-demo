package model

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	// skuRegex enforces alphanumeric characters, hyphens, and underscores between 3 and 50 characters.
	skuRegex = regexp.MustCompile(`^[A-Za-z0-9_-]{3,50}$`)
)

// Product represents a catalog product entity.
type Product struct {
	ID          int64     `json:"id"`
	SKU         string    `json:"sku"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Stock       int       `json:"stock"`
	Category    string    `json:"category"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateProductRequest defines the payload required to create a new product.
type CreateProductRequest struct {
	SKU         string  `json:"sku"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
	Category    string  `json:"category"`
}

// Validate checks whether the create request payload conforms to business rules.
func (r *CreateProductRequest) Validate() error {
	r.SKU = strings.TrimSpace(r.SKU)
	if r.SKU == "" {
		return errors.New("sku is required")
	}
	if !skuRegex.MatchString(r.SKU) {
		return errors.New("sku must be 3-50 alphanumeric characters, hyphens, or underscores")
	}

	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return errors.New("name is required")
	}
	if len(r.Name) > 200 {
		return errors.New("name must not exceed 200 characters")
	}

	r.Description = strings.TrimSpace(r.Description)
	if len(r.Description) > 2000 {
		return errors.New("description must not exceed 2000 characters")
	}

	if r.Price < 0 {
		return errors.New("price must be greater than or equal to 0")
	}

	if r.Stock < 0 {
		return errors.New("stock must be greater than or equal to 0")
	}

	r.Category = strings.TrimSpace(r.Category)
	if r.Category == "" {
		return errors.New("category is required")
	}
	if len(r.Category) > 100 {
		return errors.New("category must not exceed 100 characters")
	}

	return nil
}

// UpdateProductRequest defines the payload required to update an existing product.
type UpdateProductRequest struct {
	SKU         string  `json:"sku"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
	Category    string  `json:"category"`
}

// Validate checks whether the update request payload conforms to business rules.
func (r *UpdateProductRequest) Validate() error {
	r.SKU = strings.TrimSpace(r.SKU)
	if r.SKU == "" {
		return errors.New("sku is required")
	}
	if !skuRegex.MatchString(r.SKU) {
		return errors.New("sku must be 3-50 alphanumeric characters, hyphens, or underscores")
	}

	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return errors.New("name is required")
	}
	if len(r.Name) > 200 {
		return errors.New("name must not exceed 200 characters")
	}

	r.Description = strings.TrimSpace(r.Description)
	if len(r.Description) > 2000 {
		return errors.New("description must not exceed 2000 characters")
	}

	if r.Price < 0 {
		return errors.New("price must be greater than or equal to 0")
	}

	if r.Stock < 0 {
		return errors.New("stock must be greater than or equal to 0")
	}

	r.Category = strings.TrimSpace(r.Category)
	if r.Category == "" {
		return errors.New("category is required")
	}
	if len(r.Category) > 100 {
		return errors.New("category must not exceed 100 characters")
	}

	return nil
}

// ProductListFilter contains parameters for listing products with filtering and pagination.
type ProductListFilter struct {
	Category string
	Query    string
	Page     int
	Limit    int
}

// Normalize sanitizes filter values to safe defaults.
func (f *ProductListFilter) Normalize() {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 {
		f.Limit = 10
	} else if f.Limit > 100 {
		f.Limit = 100
	}
	f.Category = strings.TrimSpace(f.Category)
	f.Query = strings.TrimSpace(f.Query)
}

// Offset returns the SQL query offset.
func (f *ProductListFilter) Offset() int {
	return (f.Page - 1) * f.Limit
}

// Pagination contains metadata for paginated responses.
type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

// PaginatedProducts represents a paginated list of products.
type PaginatedProducts struct {
	Data       []Product  `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// NewPaginatedProducts creates a PaginatedProducts instance with calculated page count.
func NewPaginatedProducts(data []Product, totalItems int64, page, limit int) *PaginatedProducts {
	if data == nil {
		data = []Product{}
	}
	totalPages := int((totalItems + int64(limit) - 1) / int64(limit))
	if totalPages == 0 {
		totalPages = 1
	}
	return &PaginatedProducts{
		Data: data,
		Pagination: Pagination{
			Page:       page,
			Limit:      limit,
			TotalItems: totalItems,
			TotalPages: totalPages,
		},
	}
}

// String provides a human-readable representation for a Product.
func (p Product) String() string {
	return fmt.Sprintf("Product(ID=%d, SKU=%s, Name=%s, Price=%.2f, Stock=%d, Category=%s)",
		p.ID, p.SKU, p.Name, p.Price, p.Stock, p.Category)
}
