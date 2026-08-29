# Technical Design Document (TDD) — Order Management API

## 1. System Architecture

The Order Management API implements a classic 3-tier Layered Architecture (Handler -> Service -> Repository), augmented with an asynchronous Cache Layer (Redis) and Gin HTTP framework.

```
+-------------------------------------------------------------+
|                      Client Request                         |
+-------------------------------------------------------------+
                              |
                              v
+-------------------------------------------------------------+
|                      Gin HTTP Engine                        |
|  - Rate Limiter Middleware (Token Bucket / IP Window)       |
|  - Structured Request Logger                                |
|  - JWT Authentication Middleware (Bearer Tokens)            |
+-------------------------------------------------------------+
                              |
                              v
+-------------------------------------------------------------+
|                    HTTP Handler Layer                       |
|  - Request binding & Validation                             |
|  - Response formatting & Status code translation            |
+-------------------------------------------------------------+
                              |
                              v
+-------------------------------------------------------------+
|                    Service Layer (Logic)                    |
|  - Order state machine transitions & validations            |
|  - Order item total calculations                            |
|  - Cache-aside orchestration (Read-through & Invalidation)  |
+-------------------------------------------------------------+
                 /                           \
                v                             v
+-----------------------------+ +-----------------------------+
|   Repository Layer (PG)     | |      Cache Layer (Redis)    |
| - Connection pooling        | | - Customer & Order TTL cache|
| - Transaction support       | | - Fast key invalidation     |
+-----------------------------+ +-----------------------------+
                |                             |
                v                             v
       [ PostgreSQL 16 ]                 [ Redis 7 ]
```

---

## 2. Directory Layout

```
medium-api/
├── docs/
│   ├── PRD.md
│   ├── TDD.md
│   ├── API.md
│   └── postman_collection.json
├── cmd/
│   └── server/
│       └── main.go              # Application bootstrap & dependency wiring
├── internal/
│   ├── config/
│   │   └── config.go            # Environment variable configuration loader
│   ├── handler/
│   │   ├── auth.go              # Authentication handler (JWT generation)
│   │   ├── customer.go          # Customer REST handlers
│   │   ├── customer_test.go     # Customer handler tests
│   │   ├── order.go             # Order REST handlers
│   │   ├── order_test.go        # Order handler tests
│   │   └── health.go            # Health check handler
│   ├── service/
│   │   ├── customer.go          # Customer business logic & cache orchestration
│   │   ├── customer_test.go     # Customer service unit tests
│   │   ├── order.go             # Order business logic & state machine
│   │   └── order_test.go        # Order service unit tests
│   ├── repository/
│   │   ├── customer_pg.go       # PostgreSQL customer repository
│   │   ├── customer_pg_test.go  # Customer repository unit tests (sqlmock)
│   │   ├── order_pg.go          # PostgreSQL order repository with transactions
│   │   └── order_pg_test.go     # Order repository unit tests (sqlmock)
│   ├── model/
│   │   ├── customer.go          # Customer domain models & DTOs
│   │   ├── order.go             # Order & OrderItem domain models & DTOs
│   │   └── errors.go            # Custom domain & service error types
│   ├── middleware/
│   │   ├── auth.go              # JWT authentication & claims extraction
│   │   ├── logger.go            # Gin JSON structured logger
│   │   └── ratelimit.go         # Rate limiter middleware
│   └── cache/
│       ├── redis.go             # Redis cache client implementation
│       └── redis_test.go        # Cache layer unit tests
├── migrations/
│   ├── 001_create_customers.sql # Customer DDL migration
│   └── 002_create_orders.sql    # Orders and OrderItems DDL migration
├── go.mod
├── go.sum
├── Dockerfile                   # Multi-stage production container
├── docker-compose.yml           # App + PostgreSQL + Redis orchestration
├── .env.example                 # Environment variable template
└── README.md                    # Service setup & quickstart guide
```

---

## 3. Database Schema (PostgreSQL)

### 3.1 Migration 001: Customers
```sql
CREATE TABLE IF NOT EXISTS customers (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,
    phone VARCHAR(30) DEFAULT '',
    address VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_customers_email ON customers(email);
```

### 3.2 Migration 002: Orders and Order Items
```sql
CREATE TABLE IF NOT EXISTS orders (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    total_amount NUMERIC(12, 2) NOT NULL CHECK(total_amount >= 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    shipping_address VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_orders_customer_id ON orders(customer_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);

CREATE TABLE IF NOT EXISTS order_items (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL,
    product_name VARCHAR(200) NOT NULL,
    unit_price NUMERIC(12, 2) NOT NULL CHECK(unit_price >= 0),
    quantity INTEGER NOT NULL CHECK(quantity > 0),
    subtotal NUMERIC(12, 2) NOT NULL CHECK(subtotal >= 0)
);

CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);
```

---

## 4. Cache Design (Redis)

- Key Patterns:
  - `customer:{id}` -> JSON-serialized `model.Customer`, TTL: 5 minutes.
  - `order:{id}` -> JSON-serialized `model.Order` with items, TTL: 5 minutes.
- Invalidation Rules:
  - Customer updated -> `DEL customer:{id}`
  - Order status updated -> `DEL order:{id}`
  - Order cancelled -> `DEL order:{id}`

---

## 5. Security & JWT Specification

- **Algorithm**: `HS256` (HMAC SHA-256)
- **Claims**:
  - `sub`: Customer ID (e.g. `1` or `admin`)
  - `email`: User email address
  - `role`: Role (`customer` or `admin`)
  - `exp`: Token expiration time (default: 24 hours)
- **Authorization Header**: `Bearer <token>`

---

## 6. Error Handling Strategy

Standard sentinel error definitions:
- `ErrCustomerNotFound` -> HTTP 404
- `ErrOrderNotFound` -> HTTP 404
- `ErrDuplicateEmail` -> HTTP 409
- `ErrInvalidStatusTransition` -> HTTP 400
- `ErrOrderCannotBeCancelled` -> HTTP 400
- `ErrEmptyOrderItems` -> HTTP 400
