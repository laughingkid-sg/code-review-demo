package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/demo/medium-api/internal/model"
	"github.com/redis/go-redis/v9"
)

func setupTestRedis(t *testing.T) (*RedisCache, *miniredis.Miniredis) {
	t.Helper()
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(func() {
		s.Close()
	})

	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	t.Cleanup(func() {
		_ = client.Close()
	})

	return NewRedisCacheFromClient(client), s
}

func TestRedisCache_Customer_SetGetDelete(t *testing.T) {
	cache, _ := setupTestRedis(t)
	ctx := context.Background()

	customer := &model.Customer{
		ID:        1,
		Email:     "test@example.com",
		Name:      "Test Customer",
		Phone:     "123456",
		Address:   "123 Main St",
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Millisecond),
	}

	// 1. Initial Get (miss)
	got, err := cache.GetCustomer(ctx, 1)
	if err != nil || got != nil {
		t.Fatalf("expected nil on cache miss, got: %v, err: %v", got, err)
	}

	// 2. Set
	err = cache.SetCustomer(ctx, customer, time.Minute)
	if err != nil {
		t.Fatalf("failed to set customer in cache: %v", err)
	}

	// 3. Get (hit)
	got, err = cache.GetCustomer(ctx, 1)
	if err != nil || got == nil {
		t.Fatalf("expected customer on cache hit, got err: %v", err)
	}
	if got.Email != customer.Email || got.Name != customer.Name {
		t.Errorf("cached customer mismatch: %+v vs %+v", got, customer)
	}

	// 4. Delete
	err = cache.DeleteCustomer(ctx, 1)
	if err != nil {
		t.Fatalf("failed to delete customer from cache: %v", err)
	}

	// 5. Get after delete (miss)
	got, err = cache.GetCustomer(ctx, 1)
	if err != nil || got != nil {
		t.Fatalf("expected nil after delete, got: %v, err: %v", got, err)
	}
}

func TestRedisCache_Order_SetGetDelete(t *testing.T) {
	cache, _ := setupTestRedis(t)
	ctx := context.Background()

	order := &model.Order{
		ID:              100,
		CustomerID:      1,
		Status:          model.OrderStatusPending,
		TotalAmount:     89.99,
		Currency:        "USD",
		ShippingAddress: "123 Main St",
		Items: []model.OrderItem{
			{
				ID:          1,
				OrderID:     100,
				ProductID:   10,
				ProductName: "Mechanical Keyboard",
				UnitPrice:   89.99,
				Quantity:    1,
				Subtotal:    89.99,
			},
		},
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Millisecond),
	}

	// 1. Set
	err := cache.SetOrder(ctx, order, time.Minute)
	if err != nil {
		t.Fatalf("failed to set order in cache: %v", err)
	}

	// 2. Get (hit)
	got, err := cache.GetOrder(ctx, 100)
	if err != nil || got == nil {
		t.Fatalf("expected order on cache hit, got err: %v", err)
	}
	if got.ID != 100 || len(got.Items) != 1 || got.Items[0].ProductName != "Mechanical Keyboard" {
		t.Errorf("cached order mismatch: %+v vs %+v", got, order)
	}

	// 3. Delete
	err = cache.DeleteOrder(ctx, 100)
	if err != nil {
		t.Fatalf("failed to delete order: %v", err)
	}

	// 4. Get after delete
	got, err = cache.GetOrder(ctx, 100)
	if err != nil || got != nil {
		t.Fatalf("expected nil after delete, got: %v", got)
	}
}

func TestRedisCache_Ping(t *testing.T) {
	cache, _ := setupTestRedis(t)
	ctx := context.Background()

	if err := cache.Ping(ctx); err != nil {
		t.Fatalf("expected ping to succeed: %v", err)
	}
}
