package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/demo/medium-api/internal/model"
	"github.com/lib/pq"
)

// CustomerRepository defines the data persistence contract for customers.
type CustomerRepository interface {
	Create(ctx context.Context, c *model.Customer) error
	GetByID(ctx context.Context, id int64) (*model.Customer, error)
	GetByEmail(ctx context.Context, email string) (*model.Customer, error)
	List(ctx context.Context, filter model.CustomerListFilter) (*model.PaginatedCustomers, error)
	Update(ctx context.Context, c *model.Customer) error
	Delete(ctx context.Context, id int64) error
}

// CustomerPGRepository implements CustomerRepository for PostgreSQL.
type CustomerPGRepository struct {
	db *sql.DB
}

// NewCustomerPGRepository creates a new PostgreSQL customer repository.
func NewCustomerPGRepository(db *sql.DB) *CustomerPGRepository {
	return &CustomerPGRepository{db: db}
}

// Create inserts a customer and scans the generated ID and timestamps.
func (r *CustomerPGRepository) Create(ctx context.Context, c *model.Customer) error {
	now := time.Now().UTC()
	query := `
		INSERT INTO customers (email, name, phone, address, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		c.Email, c.Name, c.Phone, c.Address, now, now,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)

	if err != nil {
		if isPGUniqueViolation(err) {
			return model.ErrDuplicateEmail
		}
		return fmt.Errorf("insert customer failed: %w", err)
	}
	return nil
}

// GetByID retrieves a customer by primary key ID.
func (r *CustomerPGRepository) GetByID(ctx context.Context, id int64) (*model.Customer, error) {
	query := `
		SELECT id, email, name, phone, address, created_at, updated_at
		FROM customers
		WHERE id = $1
	`
	var c model.Customer
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID, &c.Email, &c.Name, &c.Phone, &c.Address, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrCustomerNotFound
		}
		return nil, fmt.Errorf("get customer by id %d failed: %w", id, err)
	}
	return &c, nil
}

// GetByEmail retrieves a customer by email address.
func (r *CustomerPGRepository) GetByEmail(ctx context.Context, email string) (*model.Customer, error) {
	query := `
		SELECT id, email, name, phone, address, created_at, updated_at
		FROM customers
		WHERE LOWER(email) = LOWER($1)
	`
	var c model.Customer
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&c.ID, &c.Email, &c.Name, &c.Phone, &c.Address, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrCustomerNotFound
		}
		return nil, fmt.Errorf("get customer by email %q failed: %w", email, err)
	}
	return &c, nil
}

// List returns a paginated list of customers with optional name/email text search.
func (r *CustomerPGRepository) List(ctx context.Context, filter model.CustomerListFilter) (*model.PaginatedCustomers, error) {
	filter.Normalize()

	var whereClauses []string
	var args []any
	argIdx := 1

	if filter.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(name ILIKE $%d OR email ILIKE $%d)", argIdx, argIdx))
		pattern := "%" + filter.Search + "%"
		args = append(args, pattern)
		argIdx++
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM customers" + whereSQL
	var totalItems int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return nil, fmt.Errorf("count customers failed: %w", err)
	}

	listQuery := fmt.Sprintf(`
		SELECT id, email, name, phone, address, created_at, updated_at
		FROM customers%s
		ORDER BY id ASC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	fetchArgs := make([]any, len(args), len(args)+2)
	copy(fetchArgs, args)
	fetchArgs = append(fetchArgs, filter.Limit, filter.Offset())

	rows, err := r.db.QueryContext(ctx, listQuery, fetchArgs...)
	if err != nil {
		return nil, fmt.Errorf("query customers list failed: %w", err)
	}
	defer rows.Close()

	customers := make([]model.Customer, 0, filter.Limit)
	for rows.Next() {
		var c model.Customer
		if err := rows.Scan(
			&c.ID, &c.Email, &c.Name, &c.Phone, &c.Address, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan customer row failed: %w", err)
		}
		customers = append(customers, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	return model.NewPaginatedCustomers(customers, totalItems, filter.Page, filter.Limit), nil
}

// Update modifies an existing customer record.
func (r *CustomerPGRepository) Update(ctx context.Context, c *model.Customer) error {
	now := time.Now().UTC()
	query := `
		UPDATE customers
		SET name = $1, phone = $2, address = $3, updated_at = $4
		WHERE id = $5
		RETURNING updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		c.Name, c.Phone, c.Address, now, c.ID,
	).Scan(&c.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ErrCustomerNotFound
		}
		return fmt.Errorf("update customer %d failed: %w", c.ID, err)
	}
	return nil
}

// Delete removes a customer by ID.
func (r *CustomerPGRepository) Delete(ctx context.Context, id int64) error {
	query := "DELETE FROM customers WHERE id = $1"
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete customer %d failed: %w", id, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("retrieve rows affected failed: %w", err)
	}

	if rowsAffected == 0 {
		return model.ErrCustomerNotFound
	}
	return nil
}

func isPGUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return strings.Contains(strings.ToLower(err.Error()), "unique constraint")
}
