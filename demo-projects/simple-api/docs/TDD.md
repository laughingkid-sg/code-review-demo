# Technical Design Document (TDD) — Product Catalog API

## 1. System Architecture

The Product Catalog API is designed with simplicity, zero external runtime infrastructure dependencies, and high maintainability as its core architectural principles. It utilizes the Go standard library (`net/http` router introduced in Go 1.22+) alongside a pure Go SQLite database driver (`modernc.org/sqlite`), eliminating the need for CGo compilation or external database processes.

```
+-------------------------------------------------------------+
|                      Client Request                         |
+-------------------------------------------------------------+
                              |
                              v
+-------------------------------------------------------------+
|                   HTTP Middleware Layer                     |
|  - Recovery & Request Logging                               |
|  - API Key Authentication (`X-API-Key`)                     |
+-------------------------------------------------------------+
                              |
                              v
+-------------------------------------------------------------+
|                     HTTP Handler Layer                      |
|  - Request parsing & JSON binding                           |
|  - Input validation                                         |
|  - Standardized JSON responses & Status Codes               |
+-------------------------------------------------------------+
                              |
                              v
+-------------------------------------------------------------+
|                   Store Layer (Repository)                  |
|  - Interface-based abstraction (`ProductStore`)             |
|  - SQLite implementation with parameter bindings            |
+-------------------------------------------------------------+
                              |
                              v
+-------------------------------------------------------------+
|                   SQLite Database Engine                    |
|  - WAL mode, Foreign Keys enabled                           |
|  - Auto-migrations on startup                               |
+-------------------------------------------------------------+
```

---

## 2. Directory & Package Layout

```
demo-projects/simple-api/
├── docs/
│   ├── PRD.md
│   └── TDD.md
├── cmd/
│   └── server/
│       └── main.go              # Application entrypoint & dependency injection
├── internal/
│   ├── handler/
│   │   ├── health.go            # Health check endpoint
│   │   ├── product.go           # Product REST endpoints (CRUD)
│   │   └── product_test.go      # HTTP handler unit tests
│   ├── model/
│   │   └── product.go           # Domain struct definitions & validation logic
│   ├── store/
│   │   ├── sqlite.go            # SQLite store implementation & query execution
│   │   └── sqlite_test.go       # Storage layer unit tests (using :memory:)
│   └── middleware/
│       └── apikey.go            # API key authentication middleware
├── migrations/
│   └── 001_create_products.sql  # DDL schema definition
├── go.mod
├── go.sum
├── Dockerfile                   # Multi-stage container build
├── docker-compose.yml           # Compose orchestration
├── .env.example                 # Example configuration
└── README.md                    # Quickstart guide & documentation
```

---

## 3. Data Model & Database Schema

### 3.1 SQLite DDL (`migrations/001_create_products.sql`)

```sql
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
```

### 3.2 Go Domain Model (`internal/model/product.go`)

```go
type Product struct {
    ID          int64     `json:"id"`
    SKU         string    `json:"sku"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Price       float64   `json:"price"`
    Stock       int       `json:"stock"`
    Category    string    `json:"category"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type CreateProductRequest struct {
    SKU         string  `json:"sku"`
    Name        string  `json:"name"`
    Description string  `json:"description"`
    Price       float64 `json:"price"`
    Stock       int     `json:"stock"`
    Category    string  `json:"category"`
}

type UpdateProductRequest struct {
    SKU         string  `json:"sku"`
    Name        string  `json:"name"`
    Description string  `json:"description"`
    Price       float64 `json:"price"`
    Stock       int     `json:"stock"`
    Category    string  `json:"category"`
}

type ProductListFilter struct {
    Category string
    Query    string
    Page     int
    Limit    int
}

type PaginatedProducts struct {
    Data       []Product  `json:"data"`
    Pagination Pagination `json:"pagination"`
}

type Pagination struct {
    Page       int   `json:"page"`
    Limit      int   `json:"limit"`
    TotalItems int64 `json:"total_items"`
    TotalPages int   `json:"total_pages"`
}
```

---

## 4. Storage Layer Interface

```go
type ProductStore interface {
    Create(ctx context.Context, p *model.Product) error
    GetByID(ctx context.Context, id int64) (*model.Product, error)
    GetBySKU(ctx context.Context, sku string) (*model.Product, error)
    List(ctx context.Context, filter model.ProductListFilter) (*model.PaginatedProducts, error)
    Update(ctx context.Context, p *model.Product) error
    Delete(ctx context.Context, id int64) error
    Close() error
}
```

### Error Sentinel Constants
- `store.ErrNotFound`: Returned when an entity ID is not present.
- `store.ErrDuplicateSKU`: Returned when a unique constraint on SKU is violated.

---

## 5. Middleware Specification

### API Key Authentication (`internal/middleware/apikey.go`)
- Inspects the incoming request header `X-API-Key`.
- Returns `401 Unauthorized` with `{"error": "unauthorized: invalid or missing API key"}` if the key does not match the configured secret.
- Public paths (`/api/health`, and optionally public read routes) can bypass authentication, while modification endpoints (`POST`, `PUT`, `DELETE`) enforce it strictly.

---

## 6. HTTP Routing & Endpoints (Go 1.22+ ServeMux)

| Method | Pattern | Handler Function | Auth Required |
|---|---|---|---|
| `GET` | `/api/health` | `HealthHandler` | No |
| `GET` | `/api/products` | `ListProductsHandler` | No |
| `GET` | `/api/products/{id}` | `GetProductHandler` | No |
| `POST` | `/api/products` | `CreateProductHandler` | Yes |
| `PUT` | `/api/products/{id}` | `UpdateProductHandler` | Yes |
| `DELETE` | `/api/products/{id}` | `DeleteProductHandler` | Yes |

---

## 7. SQLite Configuration & Pragmas

To ensure concurrency safety and prevent `database is locked` errors during concurrent access, the SQLite connection will be initialized with:
- `_journal_mode=WAL` (Write-Ahead Logging for non-blocking concurrent reads and single writer)
- `_busy_timeout=5000` (5-second timeout waiting for database lock)
- `_foreign_keys=ON` (Enforce relational constraints)
- `SetMaxOpenConns(1)` on write pools if single-writer constraint is needed, or pooled connections with WAL mode.

---

## 8. Configuration Parameters

| Variable | Type | Default | Description |
|---|---|---|---|
| `PORT` | string | `8080` | HTTP listening port |
| `DB_PATH` | string | `data/catalog.db` | Path to SQLite database file |
| `API_KEY` | string | `dev-api-key-12345` | API key required for protected endpoints |
| `ENV` | string | `development` | Application environment (`development`, `production`) |

---

## 9. Testing Strategy

1. **Storage Unit Tests (`internal/store/sqlite_test.go`)**:
   - Executes all CRUD operations against an in-memory SQLite database (`:memory:`).
   - Verifies constraint handling (unique SKU, not found errors, transaction rollbacks).
2. **Handler Unit Tests (`internal/handler/product_test.go`)**:
   - Table-driven tests utilizing mock stores and `httptest.ResponseRecorder`.
   - Validates status codes, JSON response formatting, query parameter handling, and error handling.
