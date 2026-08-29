package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/demo/medium-api/internal/cache"
	"github.com/demo/medium-api/internal/model"
	"github.com/demo/medium-api/internal/repository"
)

const customerCacheTTL = 5 * time.Minute

// CustomerService provides business logic for customer lifecycle and cache coordination.
type CustomerService interface {
	RegisterCustomer(ctx context.Context, req *model.CreateCustomerRequest) (*model.Customer, error)
	GetCustomerByID(ctx context.Context, id int64) (*model.Customer, error)
	ListCustomers(ctx context.Context, filter model.CustomerListFilter) (*model.PaginatedCustomers, error)
	UpdateCustomer(ctx context.Context, id int64, req *model.UpdateCustomerRequest) (*model.Customer, error)
	DeleteCustomer(ctx context.Context, id int64) error
}

// CustomerServiceImpl implements CustomerService.
type CustomerServiceImpl struct {
	repo  repository.CustomerRepository
	cache cache.Cache
}

// NewCustomerService creates a new CustomerService instance.
func NewCustomerService(repo repository.CustomerRepository, cache cache.Cache) *CustomerServiceImpl {
	return &CustomerServiceImpl{
		repo:  repo,
		cache: cache,
	}
}

// RegisterCustomer validates and persists a new customer account.
func (s *CustomerServiceImpl) RegisterCustomer(ctx context.Context, req *model.CreateCustomerRequest) (*model.Customer, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	customer := &model.Customer{
		Email:   req.Email,
		Name:    req.Name,
		Phone:   req.Phone,
		Address: req.Address,
	}

	if err := s.repo.Create(ctx, customer); err != nil {
		return nil, err
	}

	// Cache the newly created customer
	if s.cache != nil {
		if err := s.cache.SetCustomer(ctx, customer, customerCacheTTL); err != nil {
			log.Printf("failed to cache new customer %d: %v", customer.ID, err)
		}
	}

	return customer, nil
}

// GetCustomerByID retrieves customer details using cache-aside pattern.
func (s *CustomerServiceImpl) GetCustomerByID(ctx context.Context, id int64) (*model.Customer, error) {
	if id <= 0 {
		return nil, model.ErrCustomerNotFound
	}

	// 1. Attempt read from cache
	if s.cache != nil {
		if cached, err := s.cache.GetCustomer(ctx, id); err == nil && cached != nil {
			return cached, nil
		}
	}

	// 2. Query repository
	customer, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 3. Write-through to cache
	if s.cache != nil {
		if err := s.cache.SetCustomer(ctx, customer, customerCacheTTL); err != nil {
			log.Printf("failed to cache customer %d: %v", customer.ID, err)
		}
	}

	return customer, nil
}

// ListCustomers returns a paginated list of customers.
func (s *CustomerServiceImpl) ListCustomers(ctx context.Context, filter model.CustomerListFilter) (*model.PaginatedCustomers, error) {
	return s.repo.List(ctx, filter)
}

// UpdateCustomer updates customer attributes and invalidates the cached representation.
func (s *CustomerServiceImpl) UpdateCustomer(ctx context.Context, id int64, req *model.UpdateCustomerRequest) (*model.Customer, error) {
	if id <= 0 {
		return nil, model.ErrCustomerNotFound
	}
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	existing.Name = req.Name
	existing.Phone = req.Phone
	existing.Address = req.Address

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}

	// Invalidate cache
	if s.cache != nil {
		if err := s.cache.DeleteCustomer(ctx, id); err != nil {
			log.Printf("failed to evict cached customer %d: %v", id, err)
		}
	}

	return existing, nil
}

// DeleteCustomer deletes a customer and clears their cache entry.
func (s *CustomerServiceImpl) DeleteCustomer(ctx context.Context, id int64) error {
	if id <= 0 {
		return model.ErrCustomerNotFound
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	if s.cache != nil {
		_ = s.cache.DeleteCustomer(ctx, id)
	}

	return nil
}
