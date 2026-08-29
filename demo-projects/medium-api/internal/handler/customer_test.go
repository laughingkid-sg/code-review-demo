package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/demo/medium-api/internal/middleware"
	"github.com/demo/medium-api/internal/model"
	"github.com/gin-gonic/gin"
)

type mockCustomerService struct {
	customers map[int64]*model.Customer
	nextID    int64
}

func newMockCustomerService() *mockCustomerService {
	return &mockCustomerService{
		customers: make(map[int64]*model.Customer),
		nextID:    1,
	}
}

func (s *mockCustomerService) RegisterCustomer(ctx context.Context, req *model.CreateCustomerRequest) (*model.Customer, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	for _, c := range s.customers {
		if c.Email == req.Email {
			return nil, model.ErrDuplicateEmail
		}
	}
	c := &model.Customer{
		ID:        s.nextID,
		Email:     req.Email,
		Name:      req.Name,
		Phone:     req.Phone,
		Address:   req.Address,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.nextID++
	s.customers[c.ID] = c
	return c, nil
}

func (s *mockCustomerService) GetCustomerByID(ctx context.Context, id int64) (*model.Customer, error) {
	c, ok := s.customers[id]
	if !ok {
		return nil, model.ErrCustomerNotFound
	}
	return c, nil
}

func (s *mockCustomerService) ListCustomers(ctx context.Context, filter model.CustomerListFilter) (*model.PaginatedCustomers, error) {
	var list []model.Customer
	for _, c := range s.customers {
		list = append(list, *c)
	}
	return model.NewPaginatedCustomers(list, int64(len(list)), filter.Page, filter.Limit), nil
}

func (s *mockCustomerService) UpdateCustomer(ctx context.Context, id int64, req *model.UpdateCustomerRequest) (*model.Customer, error) {
	c, ok := s.customers[id]
	if !ok {
		return nil, model.ErrCustomerNotFound
	}
	c.Name = req.Name
	c.Phone = req.Phone
	c.Address = req.Address
	c.UpdatedAt = time.Now()
	return c, nil
}

func (s *mockCustomerService) DeleteCustomer(ctx context.Context, id int64) error {
	if _, ok := s.customers[id]; !ok {
		return model.ErrCustomerNotFound
	}
	delete(s.customers, id)
	return nil
}

const testJWTSecret = "test-jwt-secret-key-12345"

func setupCustomerTestRouter(svc *mockCustomerService) (*gin.Engine, string) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	authH := NewAuthHandler(svc, testJWTSecret)
	custH := NewCustomerHandler(svc)
	authMiddleware := middleware.JWTAuthMiddleware(testJWTSecret)

	r.POST("/api/auth/login", authH.Login)
	r.POST("/api/customers", custH.Register)

	protected := r.Group("/api")
	protected.Use(authMiddleware)
	{
		protected.GET("/customers", custH.List)
		protected.GET("/customers/:id", custH.Get)
		protected.PUT("/customers/:id", custH.Update)
		protected.DELETE("/customers/:id", custH.Delete)
	}

	token, _ := middleware.GenerateToken(1, "admin@demo.com", "admin", testJWTSecret, time.Hour)
	return r, token
}

func TestCustomerHandler_Register_And_Get(t *testing.T) {
	svc := newMockCustomerService()
	r, token := setupCustomerTestRouter(svc)

	t.Run("Register customer success", func(t *testing.T) {
		body := `{"email":"new@example.com","name":"New User","address":"123 St"}`
		req := httptest.NewRequest(http.MethodPost, "/api/customers", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d (body: %s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("Register duplicate email returns 409", func(t *testing.T) {
		body := `{"email":"new@example.com","name":"Duplicate","address":"123 St"}`
		req := httptest.NewRequest(http.MethodPost, "/api/customers", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict, got %d", rec.Code)
		}
	})

	t.Run("Get customer with JWT success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/customers/1", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
	})

	t.Run("Get customer unauthorized when missing token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/customers/1", nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})
}

func TestAuthHandler_Login(t *testing.T) {
	svc := newMockCustomerService()
	r, _ := setupCustomerTestRouter(svc)

	t.Run("Admin login success", func(t *testing.T) {
		body := `{"email":"admin@demo.com","password":"admin123"}`
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}

		var resp LoginResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Token == "" || resp.TokenType != "Bearer" {
			t.Errorf("invalid login response: %+v", resp)
		}
	})
}
