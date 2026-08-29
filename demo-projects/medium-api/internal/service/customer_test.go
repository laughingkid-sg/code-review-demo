package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/demo/medium-api/internal/model"
)

type mockCustomerRepo struct {
	customers map[int64]*model.Customer
	byEmail   map[string]*model.Customer
	nextID    int64
}

func newMockCustomerRepo() *mockCustomerRepo {
	return &mockCustomerRepo{
		customers: make(map[int64]*model.Customer),
		byEmail:   make(map[string]*model.Customer),
		nextID:    1,
	}
}

func (m *mockCustomerRepo) Create(ctx context.Context, c *model.Customer) error {
	if _, exists := m.byEmail[c.Email]; exists {
		return model.ErrDuplicateEmail
	}
	c.ID = m.nextID
	m.nextID++
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	m.customers[c.ID] = c
	m.byEmail[c.Email] = c
	return nil
}

func (m *mockCustomerRepo) GetByID(ctx context.Context, id int64) (*model.Customer, error) {
	c, ok := m.customers[id]
	if !ok {
		return nil, model.ErrCustomerNotFound
	}
	return c, nil
}

func (m *mockCustomerRepo) GetByEmail(ctx context.Context, email string) (*model.Customer, error) {
	c, ok := m.byEmail[email]
	if !ok {
		return nil, model.ErrCustomerNotFound
	}
	return c, nil
}

func (m *mockCustomerRepo) List(ctx context.Context, filter model.CustomerListFilter) (*model.PaginatedCustomers, error) {
	list := make([]model.Customer, 0, len(m.customers))
	for _, c := range m.customers {
		list = append(list, *c)
	}
	return model.NewPaginatedCustomers(list, int64(len(list)), filter.Page, filter.Limit), nil
}

func (m *mockCustomerRepo) Update(ctx context.Context, c *model.Customer) error {
	existing, ok := m.customers[c.ID]
	if !ok {
		return model.ErrCustomerNotFound
	}
	existing.Name = c.Name
	existing.Phone = c.Phone
	existing.Address = c.Address
	existing.UpdatedAt = time.Now()
	return nil
}

func (m *mockCustomerRepo) Delete(ctx context.Context, id int64) error {
	c, ok := m.customers[id]
	if !ok {
		return model.ErrCustomerNotFound
	}
	delete(m.customers, id)
	delete(m.byEmail, c.Email)
	return nil
}

type mockCache struct {
	customers map[int64]*model.Customer
	orders    map[int64]*model.Order
}

func newMockCache() *mockCache {
	return &mockCache{
		customers: make(map[int64]*model.Customer),
		orders:    make(map[int64]*model.Order),
	}
}

func (c *mockCache) GetCustomer(ctx context.Context, id int64) (*model.Customer, error) {
	return c.customers[id], nil
}

func (c *mockCache) SetCustomer(ctx context.Context, customer *model.Customer, ttl time.Duration) error {
	if customer != nil {
		c.customers[customer.ID] = customer
	}
	return nil
}

func (c *mockCache) DeleteCustomer(ctx context.Context, id int64) error {
	delete(c.customers, id)
	return nil
}

func (c *mockCache) GetOrder(ctx context.Context, id int64) (*model.Order, error) {
	return c.orders[id], nil
}

func (c *mockCache) SetOrder(ctx context.Context, order *model.Order, ttl time.Duration) error {
	if order != nil {
		c.orders[order.ID] = order
	}
	return nil
}

func (c *mockCache) DeleteOrder(ctx context.Context, id int64) error {
	delete(c.orders, id)
	return nil
}

func (c *mockCache) Ping(ctx context.Context) error { return nil }
func (c *mockCache) Close() error                  { return nil }

func TestCustomerService_Register_Success(t *testing.T) {
	repo := newMockCustomerRepo()
	cacheMock := newMockCache()
	svc := NewCustomerService(repo, cacheMock)

	req := &model.CreateCustomerRequest{
		Email:   "user@example.com",
		Name:    "User One",
		Phone:   "12345",
		Address: "123 Street",
	}

	cust, err := svc.RegisterCustomer(context.Background(), req)
	if err != nil {
		t.Fatalf("expected register to succeed: %v", err)
	}

	if cust.ID != 1 || cust.Email != "user@example.com" {
		t.Errorf("unexpected registered customer data: %+v", cust)
	}

	// Verify cached
	if cacheMock.customers[cust.ID] == nil {
		t.Errorf("expected customer to be cached")
	}
}

func TestCustomerService_Register_DuplicateEmail(t *testing.T) {
	repo := newMockCustomerRepo()
	svc := NewCustomerService(repo, newMockCache())

	req := &model.CreateCustomerRequest{
		Email:   "user@example.com",
		Name:    "User One",
		Address: "123 Street",
	}
	_, _ = svc.RegisterCustomer(context.Background(), req)

	_, err := svc.RegisterCustomer(context.Background(), req)
	if !errors.Is(err, model.ErrDuplicateEmail) {
		t.Fatalf("expected duplicate email error, got: %v", err)
	}
}

func TestCustomerService_GetByID_CacheHitAndMiss(t *testing.T) {
	repo := newMockCustomerRepo()
	cacheMock := newMockCache()
	svc := NewCustomerService(repo, cacheMock)

	cust, _ := svc.RegisterCustomer(context.Background(), &model.CreateCustomerRequest{
		Email:   "test@example.com",
		Name:    "Test",
		Address: "Address",
	})

	// 1. Hit cache
	fetched, err := svc.GetCustomerByID(context.Background(), cust.ID)
	if err != nil || fetched.Email != "test@example.com" {
		t.Fatalf("expected to fetch from cache: %v", err)
	}

	// 2. Clear cache to test repo fallback
	delete(cacheMock.customers, cust.ID)
	fetchedMiss, err := svc.GetCustomerByID(context.Background(), cust.ID)
	if err != nil || fetchedMiss.Email != "test@example.com" {
		t.Fatalf("expected to fetch from repo on cache miss: %v", err)
	}

	// 3. Verify re-populated in cache
	if cacheMock.customers[cust.ID] == nil {
		t.Errorf("expected cache to be repopulated")
	}
}

func TestCustomerService_Update_InvalidatesCache(t *testing.T) {
	repo := newMockCustomerRepo()
	cacheMock := newMockCache()
	svc := NewCustomerService(repo, cacheMock)

	cust, _ := svc.RegisterCustomer(context.Background(), &model.CreateCustomerRequest{
		Email:   "test@example.com",
		Name:    "Old Name",
		Address: "Address",
	})

	updated, err := svc.UpdateCustomer(context.Background(), cust.ID, &model.UpdateCustomerRequest{
		Name:    "New Name",
		Address: "New Address",
	})
	if err != nil {
		t.Fatalf("expected update to succeed: %v", err)
	}

	if updated.Name != "New Name" {
		t.Errorf("expected updated name 'New Name', got %s", updated.Name)
	}

	// Verify cache key was deleted
	if cacheMock.customers[cust.ID] != nil {
		t.Errorf("expected cache key to be evicted on update")
	}
}
