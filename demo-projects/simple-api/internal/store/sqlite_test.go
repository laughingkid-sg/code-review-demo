package store

import (
	"context"
	"errors"
	"testing"

	"github.com/demo/simple-api/internal/model"
)

func setupTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	// Use in-memory SQLite for tests
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to initialize in-memory sqlite store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	return store
}

func TestSQLiteStore_Create_Success(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	p := &model.Product{
		SKU:         "SKU-100",
		Name:        "Test Keyboard",
		Description: "A fine mechanical keyboard",
		Price:       99.99,
		Stock:       15,
		Category:    "Peripherals",
	}

	err := s.Create(ctx, p)
	if err != nil {
		t.Fatalf("expected create to succeed, got: %v", err)
	}

	if p.ID <= 0 {
		t.Errorf("expected positive product ID, got %d", p.ID)
	}
	if p.CreatedAt.IsZero() || p.UpdatedAt.IsZero() {
		t.Errorf("expected non-zero timestamps, got created=%v, updated=%v", p.CreatedAt, p.UpdatedAt)
	}

	// Verify retrieval by ID
	fetched, err := s.GetByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("expected get by ID to succeed, got: %v", err)
	}
	if fetched.SKU != p.SKU || fetched.Name != p.Name || fetched.Price != p.Price {
		t.Errorf("fetched product does not match created: %+v vs %+v", fetched, p)
	}

	// Verify retrieval by SKU
	fetchedBySKU, err := s.GetBySKU(ctx, p.SKU)
	if err != nil {
		t.Fatalf("expected get by SKU to succeed, got: %v", err)
	}
	if fetchedBySKU.ID != p.ID {
		t.Errorf("expected same ID from get by SKU, got %d vs %d", fetchedBySKU.ID, p.ID)
	}
}

func TestSQLiteStore_Create_DuplicateSKU(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	p1 := &model.Product{
		SKU:      "SKU-DUP",
		Name:     "Item 1",
		Price:    10.0,
		Stock:    5,
		Category: "General",
	}
	if err := s.Create(ctx, p1); err != nil {
		t.Fatalf("unexpected error on first create: %v", err)
	}

	p2 := &model.Product{
		SKU:      "SKU-DUP",
		Name:     "Item 2",
		Price:    20.0,
		Stock:    10,
		Category: "General",
	}
	err := s.Create(ctx, p2)
	if !errors.Is(err, ErrDuplicateSKU) {
		t.Fatalf("expected ErrDuplicateSKU, got: %v", err)
	}
}

func TestSQLiteStore_GetByID_NotFound(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	_, err := s.GetByID(ctx, 99999)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestSQLiteStore_GetBySKU_NotFound(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	_, err := s.GetBySKU(ctx, "NON-EXISTENT-SKU")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestSQLiteStore_List_FilteringAndPagination(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	products := []*model.Product{
		{SKU: "SKU-1", Name: "Gaming Mouse", Description: "RGB mouse with 16000 DPI", Price: 49.99, Stock: 50, Category: "Accessories"},
		{SKU: "SKU-2", Name: "Office Mouse", Description: "Silent wireless mouse", Price: 19.99, Stock: 100, Category: "Accessories"},
		{SKU: "SKU-3", Name: "Mechanical Keyboard", Description: "Blue switches with RGB", Price: 79.99, Stock: 30, Category: "Keyboards"},
		{SKU: "SKU-4", Name: "Ergonomic Keyboard", Description: "Split layout keyboard", Price: 119.99, Stock: 20, Category: "Keyboards"},
		{SKU: "SKU-5", Name: "4K Monitor", Description: "27 inch IPS display", Price: 299.99, Stock: 10, Category: "Monitors"},
	}

	for _, p := range products {
		if err := s.Create(ctx, p); err != nil {
			t.Fatalf("failed to insert test product: %v", err)
		}
	}

	t.Run("Default List All", func(t *testing.T) {
		res, err := s.List(ctx, model.ProductListFilter{})
		if err != nil {
			t.Fatalf("expected list to succeed, got: %v", err)
		}
		if res.Pagination.TotalItems != 5 {
			t.Errorf("expected 5 total items, got %d", res.Pagination.TotalItems)
		}
		if len(res.Data) != 5 {
			t.Errorf("expected 5 returned items, got %d", len(res.Data))
		}
	})

	t.Run("Filter by Category", func(t *testing.T) {
		res, err := s.List(ctx, model.ProductListFilter{Category: "Keyboards"})
		if err != nil {
			t.Fatalf("expected list by category to succeed, got: %v", err)
		}
		if res.Pagination.TotalItems != 2 {
			t.Errorf("expected 2 keyboards, got %d", res.Pagination.TotalItems)
		}
		for _, p := range res.Data {
			if p.Category != "Keyboards" {
				t.Errorf("expected category 'Keyboards', got %s", p.Category)
			}
		}
	})

	t.Run("Search Query", func(t *testing.T) {
		res, err := s.List(ctx, model.ProductListFilter{Query: "RGB"})
		if err != nil {
			t.Fatalf("expected search to succeed, got: %v", err)
		}
		if res.Pagination.TotalItems != 2 {
			t.Errorf("expected 2 RGB items, got %d", res.Pagination.TotalItems)
		}
	})

	t.Run("Pagination Offset & Limit", func(t *testing.T) {
		res, err := s.List(ctx, model.ProductListFilter{Page: 2, Limit: 2})
		if err != nil {
			t.Fatalf("expected paginated list to succeed, got: %v", err)
		}
		if res.Pagination.Page != 2 || res.Pagination.Limit != 2 {
			t.Errorf("unexpected pagination metadata: %+v", res.Pagination)
		}
		if res.Pagination.TotalPages != 3 {
			t.Errorf("expected 3 total pages, got %d", res.Pagination.TotalPages)
		}
		if len(res.Data) != 2 {
			t.Errorf("expected 2 items on page 2, got %d", len(res.Data))
		}
	})
}

func TestSQLiteStore_Update_Success(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	p := &model.Product{
		SKU:      "SKU-ORIG",
		Name:     "Original Name",
		Price:    50.0,
		Stock:    10,
		Category: "Test",
	}
	if err := s.Create(ctx, p); err != nil {
		t.Fatalf("failed to create initial product: %v", err)
	}

	p.Name = "Updated Name"
	p.Price = 65.50
	p.Stock = 12
	p.Description = "Added description"

	if err := s.Update(ctx, p); err != nil {
		t.Fatalf("expected update to succeed, got: %v", err)
	}

	fetched, err := s.GetByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("failed to fetch updated product: %v", err)
	}

	if fetched.Name != "Updated Name" || fetched.Price != 65.50 || fetched.Stock != 12 || fetched.Description != "Added description" {
		t.Errorf("updated fields do not match: %+v", fetched)
	}
}

func TestSQLiteStore_Update_NotFound(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	p := &model.Product{
		ID:       99999,
		SKU:      "SKU-GHOST",
		Name:     "Ghost",
		Price:    10.0,
		Stock:    1,
		Category: "Test",
	}
	err := s.Update(ctx, p)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestSQLiteStore_Update_DuplicateSKU(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	p1 := &model.Product{SKU: "SKU-A", Name: "Item A", Price: 10, Stock: 1, Category: "Test"}
	p2 := &model.Product{SKU: "SKU-B", Name: "Item B", Price: 20, Stock: 2, Category: "Test"}
	if err := s.Create(ctx, p1); err != nil {
		t.Fatalf("failed to create p1: %v", err)
	}
	if err := s.Create(ctx, p2); err != nil {
		t.Fatalf("failed to create p2: %v", err)
	}

	p2.SKU = "SKU-A"
	err := s.Update(ctx, p2)
	if !errors.Is(err, ErrDuplicateSKU) {
		t.Fatalf("expected ErrDuplicateSKU, got: %v", err)
	}
}

func TestSQLiteStore_Delete_Success(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	p := &model.Product{SKU: "SKU-DEL", Name: "Item To Delete", Price: 15, Stock: 5, Category: "Test"}
	if err := s.Create(ctx, p); err != nil {
		t.Fatalf("failed to create product: %v", err)
	}

	if err := s.Delete(ctx, p.ID); err != nil {
		t.Fatalf("expected delete to succeed, got: %v", err)
	}

	_, err := s.GetByID(ctx, p.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after deletion, got: %v", err)
	}
}

func TestSQLiteStore_Delete_NotFound(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	err := s.Delete(ctx, 99999)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestSQLiteStore_Ping(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	if err := s.Ping(ctx); err != nil {
		t.Fatalf("expected ping to succeed, got: %v", err)
	}
}
