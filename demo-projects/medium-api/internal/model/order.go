package model

import (
	"errors"
	"strings"
	"time"
)

// Order status constants
const (
	OrderStatusPending   = "PENDING"
	OrderStatusConfirmed = "CONFIRMED"
	OrderStatusShipped   = "SHIPPED"
	OrderStatusDelivered = "DELIVERED"
	OrderStatusCancelled = "CANCELLED"
)

// ValidOrderStatuses holds all allowable status strings.
var ValidOrderStatuses = map[string]bool{
	OrderStatusPending:   true,
	OrderStatusConfirmed: true,
	OrderStatusShipped:   true,
	OrderStatusDelivered: true,
	OrderStatusCancelled: true,
}

// CanTransition verifies whether a status transition is legally allowed by the state machine.
func CanTransition(from, to string) bool {
	from = strings.ToUpper(from)
	to = strings.ToUpper(to)

	if !ValidOrderStatuses[to] {
		return false
	}
	if from == to {
		return true
	}

	switch from {
	case OrderStatusPending:
		return to == OrderStatusConfirmed || to == OrderStatusCancelled
	case OrderStatusConfirmed:
		return to == OrderStatusShipped || to == OrderStatusCancelled
	case OrderStatusShipped:
		return to == OrderStatusDelivered
	case OrderStatusDelivered, OrderStatusCancelled:
		return false // Terminal states
	default:
		return false
	}
}

// OrderItem represents a specific line item in an order.
type OrderItem struct {
	ID          int64   `json:"id"`
	OrderID     int64   `json:"order_id"`
	ProductID   int64   `json:"product_id"`
	ProductName string  `json:"product_name"`
	UnitPrice   float64 `json:"unit_price"`
	Quantity    int     `json:"quantity"`
	Subtotal    float64 `json:"subtotal"`
}

// Order represents an order placed by a customer.
type Order struct {
	ID              int64       `json:"id"`
	CustomerID      int64       `json:"customer_id"`
	Status          string      `json:"status"`
	TotalAmount     float64     `json:"total_amount"`
	Currency        string      `json:"currency"`
	ShippingAddress string      `json:"shipping_address"`
	Items           []OrderItem `json:"items"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

// CreateOrderItemRequest defines the payload for an item within an order creation request.
type CreateOrderItemRequest struct {
	ProductID   int64   `json:"product_id"`
	ProductName string  `json:"product_name"`
	UnitPrice   float64 `json:"unit_price"`
	Quantity    int     `json:"quantity"`
}

// Validate checks item attributes.
func (i *CreateOrderItemRequest) Validate() error {
	if i.ProductID <= 0 {
		return errors.New("product_id must be a positive integer")
	}
	i.ProductName = strings.TrimSpace(i.ProductName)
	if i.ProductName == "" {
		return errors.New("product_name is required")
	}
	if len(i.ProductName) > 200 {
		return errors.New("product_name must not exceed 200 characters")
	}
	if i.UnitPrice < 0 {
		return errors.New("unit_price must be greater than or equal to 0")
	}
	if i.Quantity <= 0 {
		return errors.New("quantity must be at least 1")
	}
	return nil
}

// CreateOrderRequest defines the input payload for placing a new order.
type CreateOrderRequest struct {
	CustomerID      int64                    `json:"customer_id"`
	ShippingAddress string                   `json:"shipping_address"`
	Items           []CreateOrderItemRequest `json:"items"`
}

// Validate checks order structure and line items.
func (r *CreateOrderRequest) Validate() error {
	if r.CustomerID <= 0 {
		return errors.New("customer_id must be a positive integer")
	}
	r.ShippingAddress = strings.TrimSpace(r.ShippingAddress)
	if r.ShippingAddress == "" {
		return errors.New("shipping_address is required")
	}
	if len(r.ShippingAddress) > 255 {
		return errors.New("shipping_address must not exceed 255 characters")
	}
	if len(r.Items) == 0 {
		return errors.New("order must contain at least one item")
	}
	for idx, item := range r.Items {
		if err := item.Validate(); err != nil {
			return err
		}
		_ = idx
	}
	return nil
}

// UpdateOrderStatusRequest defines the payload to change order status.
type UpdateOrderStatusRequest struct {
	Status string `json:"status"`
}

// Validate ensures the target status is valid.
func (r *UpdateOrderStatusRequest) Validate() error {
	r.Status = strings.ToUpper(strings.TrimSpace(r.Status))
	if !ValidOrderStatuses[r.Status] {
		return errors.New("invalid order status")
	}
	return nil
}

// OrderListFilter encapsulates filtering options for listing orders.
type OrderListFilter struct {
	CustomerID int64
	Status     string
	Page       int
	Limit      int
}

// Normalize sanitizes order listing filters.
func (f *OrderListFilter) Normalize() {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 {
		f.Limit = 10
	} else if f.Limit > 100 {
		f.Limit = 100
	}
	f.Status = strings.ToUpper(strings.TrimSpace(f.Status))
}

// Offset returns the SQL pagination offset.
func (f *OrderListFilter) Offset() int {
	return (f.Page - 1) * f.Limit
}

// PaginatedOrders represents a paginated list of orders.
type PaginatedOrders struct {
	Data       []Order    `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// NewPaginatedOrders builds PaginatedOrders with computed total pages.
func NewPaginatedOrders(data []Order, totalItems int64, page, limit int) *PaginatedOrders {
	if data == nil {
		data = []Order{}
	}
	totalPages := int((totalItems + int64(limit) - 1) / int64(limit))
	if totalPages == 0 {
		totalPages = 1
	}
	return &PaginatedOrders{
		Data: data,
		Pagination: Pagination{
			Page:       page,
			Limit:      limit,
			TotalItems: totalItems,
			TotalPages: totalPages,
		},
	}
}

// Pagination contains metadata for paginated responses.
type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}
