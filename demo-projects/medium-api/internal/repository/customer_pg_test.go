package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/demo/medium-api/internal/model"
)

func TestCustomerPGRepository_Create_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewCustomerPGRepository(db)
	ctx := context.Background()

	now := time.Now().UTC()
	c := &model.Customer{
		Email:   "test@example.com",
		Name:    "John Doe",
		Phone:   "+1234567890",
		Address: "123 Main St",
	}

	mock.ExpectQuery(`INSERT INTO customers`).
		WithArgs(c.Email, c.Name, c.Phone, c.Address, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, now, now))

	err = repo.Create(ctx, c)
	if err != nil {
		t.Fatalf("expected create to succeed, got: %v", err)
	}

	if c.ID != 1 {
		t.Errorf("expected customer ID 1, got %d", c.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled mock expectations: %v", err)
	}
}

func TestCustomerPGRepository_GetByID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewCustomerPGRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	mock.ExpectQuery(`SELECT id, email, name, phone, address, created_at, updated_at FROM customers WHERE id = \$1`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "phone", "address", "created_at", "updated_at"}).
			AddRow(1, "alice@example.com", "Alice Smith", "+123", "456 Oak Rd", now, now))

	customer, err := repo.GetByID(ctx, 1)
	if err != nil {
		t.Fatalf("expected get by ID to succeed, got: %v", err)
	}

	if customer.Email != "alice@example.com" || customer.Name != "Alice Smith" {
		t.Errorf("unexpected customer data: %+v", customer)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled mock expectations: %v", err)
	}
}

func TestCustomerPGRepository_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewCustomerPGRepository(db)
	ctx := context.Background()

	mock.ExpectQuery(`SELECT id, email, name, phone, address, created_at, updated_at FROM customers WHERE id = \$1`).
		WithArgs(int64(999)).
		WillReturnError(sql.ErrNoRows)

	_, err = repo.GetByID(ctx, 999)
	if !errors.Is(err, model.ErrCustomerNotFound) {
		t.Fatalf("expected ErrCustomerNotFound, got: %v", err)
	}
}

func TestCustomerPGRepository_GetByEmail_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewCustomerPGRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	mock.ExpectQuery(`SELECT id, email, name, phone, address, created_at, updated_at FROM customers WHERE LOWER\(email\) = LOWER\(\$1\)`).
		WithArgs("bob@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "phone", "address", "created_at", "updated_at"}).
			AddRow(2, "bob@example.com", "Bob Jones", "", "789 Pine St", now, now))

	customer, err := repo.GetByEmail(ctx, "bob@example.com")
	if err != nil {
		t.Fatalf("expected get by email to succeed, got: %v", err)
	}

	if customer.ID != 2 {
		t.Errorf("expected customer ID 2, got %d", customer.ID)
	}
}

func TestCustomerPGRepository_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewCustomerPGRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM customers`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`SELECT id, email, name, phone, address, created_at, updated_at FROM customers ORDER BY id ASC LIMIT \$1 OFFSET \$2`).
		WithArgs(10, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "phone", "address", "created_at", "updated_at"}).
			AddRow(1, "test@example.com", "Test User", "", "Addr", now, now))

	res, err := repo.List(ctx, model.CustomerListFilter{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("expected list to succeed, got: %v", err)
	}

	if res.Pagination.TotalItems != 1 || len(res.Data) != 1 {
		t.Errorf("unexpected list result: %+v", res)
	}
}

func TestCustomerPGRepository_Update(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewCustomerPGRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	c := &model.Customer{
		ID:      1,
		Name:    "Updated Name",
		Phone:   "1112223333",
		Address: "New Address",
	}

	mock.ExpectQuery(`UPDATE customers SET name = \$1, phone = \$2, address = \$3, updated_at = \$4 WHERE id = \$5 RETURNING updated_at`).
		WithArgs(c.Name, c.Phone, c.Address, sqlmock.AnyArg(), c.ID).
		WillReturnRows(sqlmock.NewRows([]string{"updated_at"}).AddRow(now))

	err = repo.Update(ctx, c)
	if err != nil {
		t.Fatalf("expected update to succeed, got: %v", err)
	}
}

func TestCustomerPGRepository_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewCustomerPGRepository(db)
	ctx := context.Background()

	mock.ExpectExec(`DELETE FROM customers WHERE id = \$1`).
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Delete(ctx, 1)
	if err != nil {
		t.Fatalf("expected delete to succeed, got: %v", err)
	}
}
