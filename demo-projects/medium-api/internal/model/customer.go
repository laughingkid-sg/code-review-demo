package model

import (
	"errors"
	"net/mail"
	"strings"
	"time"
)

// Customer represents a registered customer account.
type Customer struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Address   string    `json:"address"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateCustomerRequest defines the input payload for customer registration.
type CreateCustomerRequest struct {
	Email   string `json:"email"`
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
}

// Validate ensures all required customer fields meet formatting and length constraints.
func (r *CreateCustomerRequest) Validate() error {
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
	if r.Email == "" {
		return errors.New("email is required")
	}
	if _, err := mail.ParseAddress(r.Email); err != nil {
		return errors.New("invalid email format")
	}

	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return errors.New("name is required")
	}
	if len(r.Name) > 150 {
		return errors.New("name must not exceed 150 characters")
	}

	r.Phone = strings.TrimSpace(r.Phone)
	if len(r.Phone) > 30 {
		return errors.New("phone must not exceed 30 characters")
	}

	r.Address = strings.TrimSpace(r.Address)
	if r.Address == "" {
		return errors.New("address is required")
	}
	if len(r.Address) > 255 {
		return errors.New("address must not exceed 255 characters")
	}

	return nil
}

// UpdateCustomerRequest defines the input payload for updating customer information.
type UpdateCustomerRequest struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
}

// Validate checks updated field values.
func (r *UpdateCustomerRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return errors.New("name is required")
	}
	if len(r.Name) > 150 {
		return errors.New("name must not exceed 150 characters")
	}

	r.Phone = strings.TrimSpace(r.Phone)
	if len(r.Phone) > 30 {
		return errors.New("phone must not exceed 30 characters")
	}

	r.Address = strings.TrimSpace(r.Address)
	if r.Address == "" {
		return errors.New("address is required")
	}
	if len(r.Address) > 255 {
		return errors.New("address must not exceed 255 characters")
	}

	return nil
}

// CustomerListFilter encapsulates query parameters for listing customers.
type CustomerListFilter struct {
	Search string
	Page   int
	Limit  int
}

// Normalize sanitizes filter values.
func (f *CustomerListFilter) Normalize() {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 {
		f.Limit = 10
	} else if f.Limit > 100 {
		f.Limit = 100
	}
	f.Search = strings.TrimSpace(f.Search)
}

// Offset returns the SQL pagination offset.
func (f *CustomerListFilter) Offset() int {
	return (f.Page - 1) * f.Limit
}

// PaginatedCustomers represents a paginated list of customers.
type PaginatedCustomers struct {
	Data       []Customer `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// NewPaginatedCustomers builds PaginatedCustomers with computed total pages.
func NewPaginatedCustomers(data []Customer, totalItems int64, page, limit int) *PaginatedCustomers {
	if data == nil {
		data = []Customer{}
	}
	totalPages := int((totalItems + int64(limit) - 1) / int64(limit))
	if totalPages == 0 {
		totalPages = 1
	}
	return &PaginatedCustomers{
		Data: data,
		Pagination: Pagination{
			Page:       page,
			Limit:      limit,
			TotalItems: totalItems,
			TotalPages: totalPages,
		},
	}
}
