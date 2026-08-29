package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/demo/medium-api/internal/model"
	"github.com/redis/go-redis/v9"
)

// Cache defines the cache storage operations for customer and order entities.
type Cache interface {
	GetCustomer(ctx context.Context, id int64) (*model.Customer, error)
	SetCustomer(ctx context.Context, customer *model.Customer, ttl time.Duration) error
	DeleteCustomer(ctx context.Context, id int64) error

	GetOrder(ctx context.Context, id int64) (*model.Order, error)
	SetOrder(ctx context.Context, order *model.Order, ttl time.Duration) error
	DeleteOrder(ctx context.Context, id int64) error

	Ping(ctx context.Context) error
	Close() error
}

// RedisCache implements Cache using a Redis client.
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache creates a new Redis cache connection.
func NewRedisCache(addr, password string, db int) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisCache{client: client}, nil
}

// NewRedisCacheFromClient creates a RedisCache from an existing client instance.
func NewRedisCacheFromClient(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

func customerKey(id int64) string {
	return fmt.Sprintf("customer:%d", id)
}

func orderKey(id int64) string {
	return fmt.Sprintf("order:%d", id)
}

// Ping checks Redis connectivity.
func (c *RedisCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Close closes the underlying Redis client pool.
func (c *RedisCache) Close() error {
	return c.client.Close()
}

// GetCustomer retrieves a cached customer by ID. Returns (nil, nil) on cache miss.
func (c *RedisCache) GetCustomer(ctx context.Context, id int64) (*model.Customer, error) {
	val, err := c.client.Get(ctx, customerKey(id)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("redis get customer failed: %w", err)
	}

	var customer model.Customer
	if err := json.Unmarshal([]byte(val), &customer); err != nil {
		return nil, fmt.Errorf("unmarshal cached customer failed: %w", err)
	}
	return &customer, nil
}

// SetCustomer writes a customer record to Redis with a TTL.
func (c *RedisCache) SetCustomer(ctx context.Context, customer *model.Customer, ttl time.Duration) error {
	if customer == nil {
		return nil
	}
	data, err := json.Marshal(customer)
	if err != nil {
		return fmt.Errorf("marshal customer failed: %w", err)
	}
	return c.client.Set(ctx, customerKey(customer.ID), data, ttl).Err()
}

// DeleteCustomer evicts a customer record from Redis.
func (c *RedisCache) DeleteCustomer(ctx context.Context, id int64) error {
	return c.client.Del(ctx, customerKey(id)).Err()
}

// GetOrder retrieves a cached order with items by ID. Returns (nil, nil) on cache miss.
func (c *RedisCache) GetOrder(ctx context.Context, id int64) (*model.Order, error) {
	val, err := c.client.Get(ctx, orderKey(id)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("redis get order failed: %w", err)
	}

	var order model.Order
	if err := json.Unmarshal([]byte(val), &order); err != nil {
		return nil, fmt.Errorf("unmarshal cached order failed: %w", err)
	}
	return &order, nil
}

// SetOrder writes an order record to Redis with a TTL.
func (c *RedisCache) SetOrder(ctx context.Context, order *model.Order, ttl time.Duration) error {
	if order == nil {
		return nil
	}
	data, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("marshal order failed: %w", err)
	}
	return c.client.Set(ctx, orderKey(order.ID), data, ttl).Err()
}

// DeleteOrder evicts an order record from Redis.
func (c *RedisCache) DeleteOrder(ctx context.Context, id int64) error {
	return c.client.Del(ctx, orderKey(id)).Err()
}
