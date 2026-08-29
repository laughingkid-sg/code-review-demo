package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/demo/medium-api/internal/model"
)

// OrderRepository defines the persistence contract for orders.
type OrderRepository interface {
	Create(ctx context.Context, o *model.Order) error
	GetByID(ctx context.Context, id int64) (*model.Order, error)
	List(ctx context.Context, filter model.OrderListFilter) (*model.PaginatedOrders, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
	Delete(ctx context.Context, id int64) error
}

// OrderPGRepository implements OrderRepository for PostgreSQL.
type OrderPGRepository struct {
	db *sql.DB
}

// NewOrderPGRepository creates a new PostgreSQL order repository.
func NewOrderPGRepository(db *sql.DB) *OrderPGRepository {
	return &OrderPGRepository{db: db}
}

// Create inserts an order and its line items inside an atomic database transaction.
func (r *OrderPGRepository) Create(ctx context.Context, o *model.Order) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction failed: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	now := time.Now().UTC()
	if o.Currency == "" {
		o.Currency = "USD"
	}
	if o.Status == "" {
		o.Status = model.OrderStatusPending
	}

	orderQuery := `
		INSERT INTO orders (customer_id, status, total_amount, currency, shipping_address, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	err = tx.QueryRowContext(ctx, orderQuery,
		o.CustomerID, o.Status, o.TotalAmount, o.Currency, o.ShippingAddress, now, now,
	).Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt)

	if err != nil {
		return fmt.Errorf("insert order header failed: %w", err)
	}

	itemQuery := `
		INSERT INTO order_items (order_id, product_id, product_name, unit_price, quantity, subtotal)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	stmt, err := tx.PrepareContext(ctx, itemQuery)
	if err != nil {
		return fmt.Errorf("prepare insert order item statement failed: %w", err)
	}
	defer stmt.Close()

	for i := range o.Items {
		item := &o.Items[i]
		item.OrderID = o.ID
		item.Subtotal = item.UnitPrice * float64(item.Quantity)

		err = stmt.QueryRowContext(ctx,
			item.OrderID, item.ProductID, item.ProductName, item.UnitPrice, item.Quantity, item.Subtotal,
		).Scan(&item.ID)

		if err != nil {
			return fmt.Errorf("insert order item failed: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit order transaction failed: %w", err)
	}
	return nil
}

// GetByID retrieves an order with all of its associated order items.
func (r *OrderPGRepository) GetByID(ctx context.Context, id int64) (*model.Order, error) {
	orderQuery := `
		SELECT id, customer_id, status, total_amount, currency, shipping_address, created_at, updated_at
		FROM orders
		WHERE id = $1
	`
	var o model.Order
	err := r.db.QueryRowContext(ctx, orderQuery, id).Scan(
		&o.ID, &o.CustomerID, &o.Status, &o.TotalAmount, &o.Currency, &o.ShippingAddress, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrOrderNotFound
		}
		return nil, fmt.Errorf("get order by id %d failed: %w", id, err)
	}

	itemsQuery := `
		SELECT id, order_id, product_id, product_name, unit_price, quantity, subtotal
		FROM order_items
		WHERE order_id = $1
		ORDER BY id ASC
	`
	rows, err := r.db.QueryContext(ctx, itemsQuery, id)
	if err != nil {
		return nil, fmt.Errorf("get order items for order %d failed: %w", id, err)
	}
	defer rows.Close()

	items := make([]model.OrderItem, 0)
	for rows.Next() {
		var item model.OrderItem
		if err := rows.Scan(
			&item.ID, &item.OrderID, &item.ProductID, &item.ProductName, &item.UnitPrice, &item.Quantity, &item.Subtotal,
		); err != nil {
			return nil, fmt.Errorf("scan order item failed: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("order items rows iteration failed: %w", err)
	}

	o.Items = items
	return &o, nil
}

// List returns a paginated list of orders matching filter criteria.
func (r *OrderPGRepository) List(ctx context.Context, filter model.OrderListFilter) (*model.PaginatedOrders, error) {
	filter.Normalize()

	var whereClauses []string
	var args []any
	argIdx := 1

	if filter.CustomerID > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("customer_id = $%d", argIdx))
		args = append(args, filter.CustomerID)
		argIdx++
	}

	if filter.Status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM orders" + whereSQL
	var totalItems int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return nil, fmt.Errorf("count orders failed: %w", err)
	}

	listQuery := fmt.Sprintf(`
		SELECT id, customer_id, status, total_amount, currency, shipping_address, created_at, updated_at
		FROM orders%s
		ORDER BY id DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	fetchArgs := make([]any, len(args), len(args)+2)
	copy(fetchArgs, args)
	fetchArgs = append(fetchArgs, filter.Limit, filter.Offset())

	rows, err := r.db.QueryContext(ctx, listQuery, fetchArgs...)
	if err != nil {
		return nil, fmt.Errorf("query orders list failed: %w", err)
	}
	defer rows.Close()

	orders := make([]model.Order, 0, filter.Limit)
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(
			&o.ID, &o.CustomerID, &o.Status, &o.TotalAmount, &o.Currency, &o.ShippingAddress, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan order row failed: %w", err)
		}
		orders = append(orders, o)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("orders rows iteration failed: %w", err)
	}

	return model.NewPaginatedOrders(orders, totalItems, filter.Page, filter.Limit), nil
}

// UpdateStatus changes the status of an existing order.
func (r *OrderPGRepository) UpdateStatus(ctx context.Context, id int64, status string) error {
	now := time.Now().UTC()
	query := `
		UPDATE orders
		SET status = $1, updated_at = $2
		WHERE id = $3
	`
	res, err := r.db.ExecContext(ctx, query, status, now, id)
	if err != nil {
		return fmt.Errorf("update order status failed: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("retrieve rows affected failed: %w", err)
	}

	if rowsAffected == 0 {
		return model.ErrOrderNotFound
	}
	return nil
}

// Delete removes an order by ID (cascades to order_items).
func (r *OrderPGRepository) Delete(ctx context.Context, id int64) error {
	query := "DELETE FROM orders WHERE id = $1"
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete order %d failed: %w", id, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("retrieve rows affected failed: %w", err)
	}

	if rowsAffected == 0 {
		return model.ErrOrderNotFound
	}
	return nil
}
