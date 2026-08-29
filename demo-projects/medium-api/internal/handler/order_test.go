package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/demo/medium-api/internal/middleware"
	"github.com/demo/medium-api/internal/model"
	"github.com/gin-gonic/gin"
)

type mockOrderService struct {
	orders map[int64]*model.Order
	nextID int64
}

func newMockOrderService() *mockOrderService {
	return &mockOrderService{
		orders: make(map[int64]*model.Order),
		nextID: 1,
	}
}

func (s *mockOrderService) CreateOrder(ctx context.Context, req *model.CreateOrderRequest) (*model.Order, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if req.CustomerID == 999 {
		return nil, model.ErrCustomerNotFound
	}
	var total float64
	items := make([]model.OrderItem, len(req.Items))
	for i, item := range req.Items {
		sub := item.UnitPrice * float64(item.Quantity)
		total += sub
		items[i] = model.OrderItem{
			ID:          int64(i + 1),
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			UnitPrice:   item.UnitPrice,
			Quantity:    item.Quantity,
			Subtotal:    sub,
		}
	}
	order := &model.Order{
		ID:              s.nextID,
		CustomerID:      req.CustomerID,
		Status:          model.OrderStatusPending,
		TotalAmount:     total,
		Currency:        "USD",
		ShippingAddress: req.ShippingAddress,
		Items:           items,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	s.nextID++
	s.orders[order.ID] = order
	return order, nil
}

func (s *mockOrderService) GetOrderByID(ctx context.Context, id int64) (*model.Order, error) {
	o, ok := s.orders[id]
	if !ok {
		return nil, model.ErrOrderNotFound
	}
	return o, nil
}

func (s *mockOrderService) ListOrders(ctx context.Context, filter model.OrderListFilter) (*model.PaginatedOrders, error) {
	var list []model.Order
	for _, o := range s.orders {
		list = append(list, *o)
	}
	return model.NewPaginatedOrders(list, int64(len(list)), filter.Page, filter.Limit), nil
}

func (s *mockOrderService) UpdateOrderStatus(ctx context.Context, id int64, req *model.UpdateOrderStatusRequest) (*model.Order, error) {
	o, ok := s.orders[id]
	if !ok {
		return nil, model.ErrOrderNotFound
	}
	if !model.CanTransition(o.Status, req.Status) {
		return nil, model.ErrInvalidStatusTransition
	}
	o.Status = req.Status
	o.UpdatedAt = time.Now()
	return o, nil
}

func (s *mockOrderService) CancelOrder(ctx context.Context, id int64) (*model.Order, error) {
	o, ok := s.orders[id]
	if !ok {
		return nil, model.ErrOrderNotFound
	}
	if o.Status != model.OrderStatusPending && o.Status != model.OrderStatusConfirmed {
		return nil, model.ErrOrderCannotBeCancelled
	}
	o.Status = model.OrderStatusCancelled
	o.UpdatedAt = time.Now()
	return o, nil
}

func setupOrderTestRouter(svc *mockOrderService) (*gin.Engine, string) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	orderH := NewOrderHandler(svc)
	authMiddleware := middleware.JWTAuthMiddleware(testJWTSecret)

	protected := r.Group("/api")
	protected.Use(authMiddleware)
	{
		protected.POST("/orders", orderH.Create)
		protected.GET("/orders", orderH.List)
		protected.GET("/orders/:id", orderH.Get)
		protected.PATCH("/orders/:id/status", orderH.UpdateStatus)
		protected.DELETE("/orders/:id", orderH.Cancel)
	}

	token, _ := middleware.GenerateToken(1, "test@demo.com", "customer", testJWTSecret, time.Hour)
	return r, token
}

func TestOrderHandler_Create_And_Get(t *testing.T) {
	svc := newMockOrderService()
	r, token := setupOrderTestRouter(svc)

	t.Run("Create order success", func(t *testing.T) {
		body := `{
			"customer_id": 1,
			"shipping_address": "123 Broadway",
			"items": [
				{"product_id": 10, "product_name": "Mechanical Keyboard", "unit_price": 89.99, "quantity": 1}
			]
		}`
		req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d (body: %s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("Get order by ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/orders/1", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
	})

	t.Run("Update status CONFIRMED", func(t *testing.T) {
		body := `{"status":"CONFIRMED"}`
		req := httptest.NewRequest(http.MethodPatch, "/api/orders/1/status", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
	})

	t.Run("Cancel order", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/orders/1", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
	})
}
