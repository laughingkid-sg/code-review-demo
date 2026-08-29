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

func TestOrderPGRepository_Create_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewOrderPGRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	order := &model.Order{
		CustomerID:      1,
		Status:          model.OrderStatusPending,
		TotalAmount:     120.50,
		Currency:        "USD",
		ShippingAddress: "123 Market St",
		Items: []model.OrderItem{
			{
				ProductID:   10,
				ProductName: "Mechanical Keyboard",
				UnitPrice:   120.50,
				Quantity:    1,
				Subtotal:    120.50,
			},
		},
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO orders`).
		WithArgs(order.CustomerID, order.Status, order.TotalAmount, order.Currency, order.ShippingAddress, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(100, now, now))

	mock.ExpectPrepare(`INSERT INTO order_items`)
	mock.ExpectQuery(`INSERT INTO order_items`).
		WithArgs(int64(100), int64(10), "Mechanical Keyboard", 120.50, 1, 120.50).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	mock.ExpectCommit()

	err = repo.Create(ctx, order)
	if err != nil {
		t.Fatalf("expected order create to succeed, got: %v", err)
	}

	if order.ID != 100 {
		t.Errorf("expected order ID 100, got %d", order.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled mock expectations: %v", err)
	}
}

func TestOrderPGRepository_GetByID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewOrderPGRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	mock.ExpectQuery(`SELECT id, customer_id, status, total_amount, currency, shipping_address, created_at, updated_at FROM orders WHERE id = \$1`).
		WithArgs(int64(100)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "customer_id", "status", "total_amount", "currency", "shipping_address", "created_at", "updated_at"}).
			AddRow(100, 1, model.OrderStatusPending, 120.50, "USD", "123 Market St", now, now))

	mock.ExpectQuery(`SELECT id, order_id, product_id, product_name, unit_price, quantity, subtotal FROM order_items WHERE order_id = \$1`).
		WithArgs(int64(100)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "order_id", "product_id", "product_name", "unit_price", "quantity", "subtotal"}).
			AddRow(1, 100, 10, "Mechanical Keyboard", 120.50, 1, 120.50))

	order, err := repo.GetByID(ctx, 100)
	if err != nil {
		t.Fatalf("expected get order by ID to succeed, got: %v", err)
	}

	if order.ID != 100 || len(order.Items) != 1 {
		t.Errorf("unexpected order data: %+v", order)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled mock expectations: %v", err)
	}
}

func TestOrderPGRepository_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewOrderPGRepository(db)
	ctx := context.Background()

	mock.ExpectQuery(`SELECT id, customer_id, status, total_amount, currency, shipping_address, created_at, updated_at FROM orders WHERE id = \$1`).
		WithArgs(int64(999)).
		WillReturnError(sql.ErrNoRows)

	_, err = repo.GetByID(ctx, 999)
	if !errors.Is(err, model.ErrOrderNotFound) {
		t.Fatalf("expected ErrOrderNotFound, got: %v", err)
	}
}

func TestOrderPGRepository_UpdateStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewOrderPGRepository(db)
	ctx := context.Background()

	mock.ExpectExec(`UPDATE orders SET status = \$1, updated_at = \$2 WHERE id = \$3`).
		WithArgs(model.OrderStatusConfirmed, sqlmock.AnyArg(), int64(100)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.UpdateStatus(ctx, 100, model.OrderStatusConfirmed)
	if err != nil {
		t.Fatalf("expected update status to succeed, got: %v", err)
	}
}

func TestOrderPGRepository_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewOrderPGRepository(db)
	ctx := context.Background()

	mock.ExpectExec(`DELETE FROM orders WHERE id = \$1`).
		WithArgs(int64(100)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Delete(ctx, 100)
	if err != nil {
		t.Fatalf("expected delete order to succeed, got: %v", err)
	}
}
