package service

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/demo/medium-api/internal/cache"
	"github.com/demo/medium-api/internal/model"
	"github.com/demo/medium-api/internal/repository"
)

const orderCacheTTL = 5 * time.Minute

// OrderService provides business logic for order processing and state machine management.
type OrderService interface {
	CreateOrder(ctx context.Context, req *model.CreateOrderRequest) (*model.Order, error)
	GetOrderByID(ctx context.Context, id int64) (*model.Order, error)
	ListOrders(ctx context.Context, filter model.OrderListFilter) (*model.PaginatedOrders, error)
	UpdateOrderStatus(ctx context.Context, id int64, req *model.UpdateOrderStatusRequest) (*model.Order, error)
	CancelOrder(ctx context.Context, id int64) (*model.Order, error)
}

// OrderServiceImpl implements OrderService.
type OrderServiceImpl struct {
	orderRepo    repository.OrderRepository
	customerRepo repository.CustomerRepository
	cache        cache.Cache
}

// NewOrderService creates a new OrderService instance.
func NewOrderService(
	orderRepo repository.OrderRepository,
	customerRepo repository.CustomerRepository,
	cache cache.Cache,
) *OrderServiceImpl {
	return &OrderServiceImpl{
		orderRepo:    orderRepo,
		customerRepo: customerRepo,
		cache:        cache,
	}
}

// CreateOrder validates input, verifies customer existence, calculates line items and total, and persists the order.
func (s *OrderServiceImpl) CreateOrder(ctx context.Context, req *model.CreateOrderRequest) (*model.Order, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Verify customer existence
	if s.customerRepo != nil {
		if _, err := s.customerRepo.GetByID(ctx, req.CustomerID); err != nil {
			return nil, err
		}
	}

	var totalAmount float64
	items := make([]model.OrderItem, len(req.Items))
	for i, itemReq := range req.Items {
		subtotal := itemReq.UnitPrice * float64(itemReq.Quantity)
		totalAmount += subtotal
		items[i] = model.OrderItem{
			ProductID:   itemReq.ProductID,
			ProductName: itemReq.ProductName,
			UnitPrice:   itemReq.UnitPrice,
			Quantity:    itemReq.Quantity,
			Subtotal:    subtotal,
		}
	}

	order := &model.Order{
		CustomerID:      req.CustomerID,
		Status:          model.OrderStatusPending,
		TotalAmount:     totalAmount,
		Currency:        "USD",
		ShippingAddress: req.ShippingAddress,
		Items:           items,
	}

	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, err
	}

	// Cache new order
	if s.cache != nil {
		if err := s.cache.SetOrder(ctx, order, orderCacheTTL); err != nil {
			log.Printf("failed to cache new order %d: %v", order.ID, err)
		}
	}

	return order, nil
}

// GetOrderByID retrieves an order with items using cache-aside pattern.
func (s *OrderServiceImpl) GetOrderByID(ctx context.Context, id int64) (*model.Order, error) {
	if id <= 0 {
		return nil, model.ErrOrderNotFound
	}

	// 1. Check cache
	if s.cache != nil {
		if cached, err := s.cache.GetOrder(ctx, id); err == nil && cached != nil {
			return cached, nil
		}
	}

	// 2. Query repository
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 3. Populate cache
	if s.cache != nil {
		if err := s.cache.SetOrder(ctx, order, orderCacheTTL); err != nil {
			log.Printf("failed to cache order %d: %v", order.ID, err)
		}
	}

	return order, nil
}

// ListOrders returns a paginated list of orders.
func (s *OrderServiceImpl) ListOrders(ctx context.Context, filter model.OrderListFilter) (*model.PaginatedOrders, error) {
	return s.orderRepo.List(ctx, filter)
}

// UpdateOrderStatus applies state transition validation before altering order state and invalidating cache.
func (s *OrderServiceImpl) UpdateOrderStatus(ctx context.Context, id int64, req *model.UpdateOrderStatusRequest) (*model.Order, error) {
	if id <= 0 {
		return nil, model.ErrOrderNotFound
	}
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	existing, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	targetStatus := strings.ToUpper(strings.TrimSpace(req.Status))
	if !model.CanTransition(existing.Status, targetStatus) {
		return nil, model.ErrInvalidStatusTransition
	}

	if err := s.orderRepo.UpdateStatus(ctx, id, targetStatus); err != nil {
		return nil, err
	}

	// Evict cached order
	if s.cache != nil {
		if err := s.cache.DeleteOrder(ctx, id); err != nil {
			log.Printf("failed to evict cached order %d: %v", id, err)
		}
	}

	existing.Status = targetStatus
	existing.UpdatedAt = time.Now().UTC()
	return existing, nil
}

// CancelOrder transitions an eligible order to CANCELLED state.
func (s *OrderServiceImpl) CancelOrder(ctx context.Context, id int64) (*model.Order, error) {
	if id <= 0 {
		return nil, model.ErrOrderNotFound
	}

	existing, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if existing.Status != model.OrderStatusPending && existing.Status != model.OrderStatusConfirmed {
		return nil, model.ErrOrderCannotBeCancelled
	}

	if err := s.orderRepo.UpdateStatus(ctx, id, model.OrderStatusCancelled); err != nil {
		return nil, err
	}

	if s.cache != nil {
		_ = s.cache.DeleteOrder(ctx, id)
	}

	existing.Status = model.OrderStatusCancelled
	existing.UpdatedAt = time.Now().UTC()
	return existing, nil
}
