package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/demo/medium-api/internal/model"
)

type mockOrderRepo struct {
	orders map[int64]*model.Order
	nextID int64
}

func newMockOrderRepo() *mockOrderRepo {
	return &mockOrderRepo{
		orders: make(map[int64]*model.Order),
		nextID: 1,
	}
}

func (m *mockOrderRepo) Create(ctx context.Context, o *model.Order) error {
	o.ID = m.nextID
	m.nextID++
	o.CreatedAt = time.Now()
	o.UpdatedAt = time.Now()
	m.orders[o.ID] = o
	return nil
}

func (m *mockOrderRepo) GetByID(ctx context.Context, id int64) (*model.Order, error) {
	o, ok := m.orders[id]
	if !ok {
		return nil, model.ErrOrderNotFound
	}
	return o, nil
}

func (m *mockOrderRepo) List(ctx context.Context, filter model.OrderListFilter) (*model.PaginatedOrders, error) {
	list := make([]model.Order, 0, len(m.orders))
	for _, o := range m.orders {
		list = append(list, *o)
	}
	return model.NewPaginatedOrders(list, int64(len(list)), filter.Page, filter.Limit), nil
}

func (m *mockOrderRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	o, ok := m.orders[id]
	if !ok {
		return model.ErrOrderNotFound
	}
	o.Status = status
	o.UpdatedAt = time.Now()
	return nil
}

func (m *mockOrderRepo) Delete(ctx context.Context, id int64) error {
	if _, ok := m.orders[id]; !ok {
		return model.ErrOrderNotFound
	}
	delete(m.orders, id)
	return nil
}

func TestOrderService_CreateOrder_Success(t *testing.T) {
	orderRepo := newMockOrderRepo()
	custRepo := newMockCustomerRepo()
	cacheMock := newMockCache()
	svc := NewOrderService(orderRepo, custRepo, cacheMock)

	// Seed customer
	_ = custRepo.Create(context.Background(), &model.Customer{
		Email:   "buyer@example.com",
		Name:    "Buyer",
		Address: "123 Street",
	})

	req := &model.CreateOrderRequest{
		CustomerID:      1,
		ShippingAddress: "123 Street",
		Items: []model.CreateOrderItemRequest{
			{
				ProductID:   10,
				ProductName: "Item A",
				UnitPrice:   50.0,
				Quantity:    2,
			},
			{
				ProductID:   20,
				ProductName: "Item B",
				UnitPrice:   25.0,
				Quantity:    1,
			},
		},
	}

	order, err := svc.CreateOrder(context.Background(), req)
	if err != nil {
		t.Fatalf("expected create order to succeed: %v", err)
	}

	if order.ID != 1 || order.TotalAmount != 125.0 || order.Status != model.OrderStatusPending {
		t.Errorf("unexpected order values: %+v", order)
	}

	if cacheMock.orders[order.ID] == nil {
		t.Errorf("expected order to be cached")
	}
}

func TestOrderService_CreateOrder_CustomerNotFound(t *testing.T) {
	orderRepo := newMockOrderRepo()
	custRepo := newMockCustomerRepo()
	svc := NewOrderService(orderRepo, custRepo, newMockCache())

	req := &model.CreateOrderRequest{
		CustomerID:      999,
		ShippingAddress: "123 Street",
		Items: []model.CreateOrderItemRequest{
			{ProductID: 1, ProductName: "Item", UnitPrice: 10, Quantity: 1},
		},
	}

	_, err := svc.CreateOrder(context.Background(), req)
	if !errors.Is(err, model.ErrCustomerNotFound) {
		t.Fatalf("expected ErrCustomerNotFound, got: %v", err)
	}
}

func TestOrderService_UpdateOrderStatus_StateTransitions(t *testing.T) {
	orderRepo := newMockOrderRepo()
	custRepo := newMockCustomerRepo()
	cacheMock := newMockCache()
	svc := NewOrderService(orderRepo, custRepo, cacheMock)

	_ = custRepo.Create(context.Background(), &model.Customer{ID: 1, Email: "a@b.com", Name: "A", Address: "Addr"})

	order, _ := svc.CreateOrder(context.Background(), &model.CreateOrderRequest{
		CustomerID:      1,
		ShippingAddress: "Addr",
		Items:           []model.CreateOrderItemRequest{{ProductID: 1, ProductName: "X", UnitPrice: 10, Quantity: 1}},
	})

	t.Run("Valid transition PENDING -> CONFIRMED", func(t *testing.T) {
		updated, err := svc.UpdateOrderStatus(context.Background(), order.ID, &model.UpdateOrderStatusRequest{
			Status: model.OrderStatusConfirmed,
		})
		if err != nil {
			t.Fatalf("expected transition to succeed: %v", err)
		}
		if updated.Status != model.OrderStatusConfirmed {
			t.Errorf("expected CONFIRMED, got %s", updated.Status)
		}
	})

	t.Run("Valid transition CONFIRMED -> SHIPPED", func(t *testing.T) {
		updated, err := svc.UpdateOrderStatus(context.Background(), order.ID, &model.UpdateOrderStatusRequest{
			Status: model.OrderStatusShipped,
		})
		if err != nil {
			t.Fatalf("expected transition to succeed: %v", err)
		}
		if updated.Status != model.OrderStatusShipped {
			t.Errorf("expected SHIPPED, got %s", updated.Status)
		}
	})

	t.Run("Invalid transition SHIPPED -> PENDING", func(t *testing.T) {
		_, err := svc.UpdateOrderStatus(context.Background(), order.ID, &model.UpdateOrderStatusRequest{
			Status: model.OrderStatusPending,
		})
		if !errors.Is(err, model.ErrInvalidStatusTransition) {
			t.Fatalf("expected ErrInvalidStatusTransition, got: %v", err)
		}
	})
}

func TestOrderService_CancelOrder(t *testing.T) {
	orderRepo := newMockOrderRepo()
	custRepo := newMockCustomerRepo()
	svc := NewOrderService(orderRepo, custRepo, newMockCache())

	_ = custRepo.Create(context.Background(), &model.Customer{ID: 1, Email: "a@b.com", Name: "A", Address: "Addr"})

	t.Run("Cancel PENDING order succeeds", func(t *testing.T) {
		order, _ := svc.CreateOrder(context.Background(), &model.CreateOrderRequest{
			CustomerID:      1,
			ShippingAddress: "Addr",
			Items:           []model.CreateOrderItemRequest{{ProductID: 1, ProductName: "X", UnitPrice: 10, Quantity: 1}},
		})

		cancelled, err := svc.CancelOrder(context.Background(), order.ID)
		if err != nil {
			t.Fatalf("expected cancel to succeed: %v", err)
		}
		if cancelled.Status != model.OrderStatusCancelled {
			t.Errorf("expected CANCELLED status, got %s", cancelled.Status)
		}
	})

	t.Run("Cancel SHIPPED order fails", func(t *testing.T) {
		order, _ := svc.CreateOrder(context.Background(), &model.CreateOrderRequest{
			CustomerID:      1,
			ShippingAddress: "Addr",
			Items:           []model.CreateOrderItemRequest{{ProductID: 1, ProductName: "X", UnitPrice: 10, Quantity: 1}},
		})
		_, _ = svc.UpdateOrderStatus(context.Background(), order.ID, &model.UpdateOrderStatusRequest{Status: model.OrderStatusConfirmed})
		_, _ = svc.UpdateOrderStatus(context.Background(), order.ID, &model.UpdateOrderStatusRequest{Status: model.OrderStatusShipped})

		_, err := svc.CancelOrder(context.Background(), order.ID)
		if !errors.Is(err, model.ErrOrderCannotBeCancelled) {
			t.Fatalf("expected ErrOrderCannotBeCancelled, got: %v", err)
		}
	})
}
