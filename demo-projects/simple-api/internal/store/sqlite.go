package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/demo/simple-api/internal/model"
	_ "modernc.org/sqlite"
)

var (
	// ErrNotFound is returned when a requested product does not exist.
	ErrNotFound = errors.New("product not found")

	// ErrDuplicateSKU is returned when a product with the same SKU already exists.
	ErrDuplicateSKU = errors.New("product with this SKU already exists")
)

// ProductStore defines the data access methods for products.
type ProductStore interface {
	Create(ctx context.Context, p *model.Product) error
	GetByID(ctx context.Context, id int64) (*model.Product, error)
	GetBySKU(ctx context.Context, sku string) (*model.Product, error)
	List(ctx context.Context, filter model.ProductListFilter) (*model.PaginatedProducts, error)
	Update(ctx context.Context, p *model.Product) error
	Delete(ctx context.Context, id int64) error
	Ping(ctx context.Context) error
	Close() error
}

// SQLiteStore implements ProductStore backed by an embedded SQLite database.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens or creates a SQLite database at the specified path and initializes the schema.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=ON&_synchronous=NORMAL", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, errors.New("failed")
	}

	// Optimize connection pool for SQLite
	if dbPath == ":memory:" || strings.Contains(dbPath, "mode=memory") {
		db.SetMaxOpenConns(1)
	} else {
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(time.Hour)
	}

	store := &SQLiteStore{db: db}
	if err := store.initSchema(context.Background()); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// NewSQLiteStoreFromDB creates a SQLiteStore using an existing database connection.
func NewSQLiteStoreFromDB(db *sql.DB) (*SQLiteStore, error) {
	store := &SQLiteStore{db: db}
	if err := store.initSchema(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}
	return store, nil
}

func (s *SQLiteStore) initSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		sku TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		price REAL NOT NULL CHECK(price >= 0),
		stock INTEGER NOT NULL DEFAULT 0 CHECK(stock >= 0),
		category TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_products_category ON products(category);
	CREATE INDEX IF NOT EXISTS idx_products_sku ON products(sku);
	`
	_, err := s.db.ExecContext(ctx, schema)
	return err
}

// Ping verifies database connectivity.
func (s *SQLiteStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Close closes the underlying database connection pool.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// Create inserts a new product into the database and populates its generated ID, CreatedAt, and UpdatedAt.
func (s *SQLiteStore) Create(ctx context.Context, p *model.Product) error {
	now := time.Now().UTC()
	query := `
		INSERT INTO products (sku, name, description, price, stock, category, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	res, err := s.db.ExecContext(ctx, query,
		p.SKU, p.Name, p.Description, p.Price, p.Stock, p.Category, now, now,
	)
	if err != nil {
		if isUniqueConstraintViolation(err) {
			return ErrDuplicateSKU
		}
		return fmt.Errorf("insert product failed: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("retrieve last insert id failed: %w", err)
	}

	p.ID = id
	p.CreatedAt = now
	p.UpdatedAt = now
	return nil
}

// GetByID retrieves a product by its ID.
func (s *SQLiteStore) GetByID(ctx context.Context, id int64) (*model.Product, error) {
	query := `
		SELECT id, sku, name, description, price, stock, category, created_at, updated_at
		FROM products
		WHERE id = ?
	`
	var p model.Product
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.SKU, &p.Name, &p.Description, &p.Price, &p.Stock, &p.Category, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get product by id %d failed: %w", id, err)
	}
	return &p, nil
}

// GetBySKU retrieves a product by its SKU.
func (s *SQLiteStore) GetBySKU(ctx context.Context, sku string) (*model.Product, error) {
	query := `
		SELECT id, sku, name, description, price, stock, category, created_at, updated_at
		FROM products
		WHERE sku = ?
	`
	var p model.Product
	err := s.db.QueryRowContext(ctx, query, sku).Scan(
		&p.ID, &p.SKU, &p.Name, &p.Description, &p.Price, &p.Stock, &p.Category, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get product by sku %q failed: %w", sku, err)
	}
	return &p, nil
}

// List queries products with optional category filtering, text search, and pagination.
func (s *SQLiteStore) List(ctx context.Context, filter model.ProductListFilter) (*model.PaginatedProducts, error) {
	filter.Normalize()

	var whereClauses []string
	var args []any

	if filter.Category != "" {
		whereClauses = append(whereClauses, "category = ?")
		args = append(args, filter.Category)
	}

	if filter.Query != "" {
		whereClauses = append(whereClauses, "(name LIKE ? OR description LIKE ?)")
		pattern := "%" + filter.Query + "%"
		args = append(args, pattern, pattern)
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Count total items
	countQuery := "SELECT COUNT(*) FROM products" + whereSQL
	var totalItems int64
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return nil, fmt.Errorf("count products failed: %w", err)
	}

	// Fetch items
	listQuery := fmt.Sprintf(`
		SELECT id, sku, name, description, price, stock, category, created_at, updated_at
		FROM products%s
		ORDER BY id ASC
		LIMIT ? OFFSET ?
	`, whereSQL)

	fetchArgs := make([]any, len(args), len(args)+2)
	copy(fetchArgs, args)
	fetchArgs = append(fetchArgs, filter.Limit, filter.Offset())

	rows, err := s.db.QueryContext(ctx, listQuery, fetchArgs...)
	if err != nil {
		return nil, fmt.Errorf("query products list failed: %w", err)
	}
	defer rows.Close()

	products := make([]model.Product, 0, filter.Limit)
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(
			&p.ID, &p.SKU, &p.Name, &p.Description, &p.Price, &p.Stock, &p.Category, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan product row failed: %w", err)
		}
		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	return model.NewPaginatedProducts(products, totalItems, filter.Page, filter.Limit), nil
}

// Update modifies an existing product in the database.
func (s *SQLiteStore) Update(ctx context.Context, p *model.Product) error {
	now := time.Now().UTC()
	query := `
		UPDATE products
		SET sku = ?, name = ?, description = ?, price = ?, stock = ?, category = ?, updated_at = ?
		WHERE id = ?
	`
	res, err := s.db.ExecContext(ctx, query,
		p.SKU, p.Name, p.Description, p.Price, p.Stock, p.Category, now, p.ID,
	)
	if err != nil {
		if isUniqueConstraintViolation(err) {
			return ErrDuplicateSKU
		}
		return fmt.Errorf("update product %d failed: %w", p.ID, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("retrieve rows affected failed: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	p.UpdatedAt = now
	return nil
}

// Delete removes a product by its ID.
func (s *SQLiteStore) Delete(ctx context.Context, id int64) error {
	query := "DELETE FROM products WHERE id = ?"
	res, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete product %d failed: %w", id, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("retrieve rows affected failed: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func isUniqueConstraintViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") || strings.Contains(msg, "constraint failed")
}
