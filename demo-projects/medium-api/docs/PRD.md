# Product Requirements Document (PRD) — Order Management API

## 1. Executive Summary

The Order Management API is an intermediate-tier microservice designed to handle customer accounts, shopping order placements, inventory tracking associations, and order lifecycle transitions within an e-commerce platform. It demonstrates a layered architecture with JWT authentication, caching with Redis, persistence with PostgreSQL, and rate limiting.

## 2. Personas & Use Cases

- **Customer / Shopper**:
  - Registers an account and authenticates to obtain a JWT session token.
  - Places new orders containing one or more items.
  - Views order history and order delivery status.
  - Cancels pending orders before shipment.
- **Operations & Fulfillment Admin**:
  - Reviews incoming orders across all customers.
  - Updates order status through state transitions (`PENDING` -> `CONFIRMED` -> `SHIPPED` -> `DELIVERED`).
  - Manages customer profile records.

---

## 3. Core Entities & State Machine

### 3.1 Customer Entity
| Field | Type | Constraints | Description |
|---|---|---|---|
| `id` | UUID / Int64 | Primary Key | Unique customer ID |
| `email` | String | Unique, Valid Email, Required | Login email identifier |
| `name` | String | Required, 1-150 chars | Full customer name |
| `phone` | String | Optional, 1-30 chars | Contact phone number |
| `address` | String | Required, 1-255 chars | Primary shipping address |
| `created_at` | Timestamp (UTC) | Default `NOW()` | Registration timestamp |
| `updated_at` | Timestamp (UTC) | Default `NOW()` | Last update timestamp |

### 3.2 Order & OrderItem Entities
| Field | Type | Constraints | Description |
|---|---|---|---|
| `id` | Int64 / UUID | Primary Key | Unique order identifier |
| `customer_id` | Int64 / UUID | Foreign Key -> `customers.id` | Associated customer |
| `status` | String Enum | `PENDING`, `CONFIRMED`, `SHIPPED`, `DELIVERED`, `CANCELLED` | Current order state |
| `total_amount` | Decimal / Float | >= 0.00 | Calculated sum of all items |
| `currency` | String | Default `USD` | ISO 4217 Currency Code |
| `shipping_address`| String | Required | Shipping destination address |
| `items` | Array of `OrderItem` | Min 1 item required | Line items in the order |
| `created_at` | Timestamp (UTC) | Default `NOW()` | Order placement time |
| `updated_at` | Timestamp (UTC) | Default `NOW()` | Last update timestamp |

#### Order Item Fields:
- `id` (int64)
- `order_id` (int64)
- `product_id` (int64/string)
- `product_name` (string)
- `unit_price` (float >= 0)
- `quantity` (int >= 1)
- `subtotal` (float = `unit_price * quantity`)

### 3.3 Order Lifecycle State Transitions
```
                +-------------------+
                |      PENDING      |
                +-------------------+
                  /               \
                 /                 \
                v                   v
     +-------------------+   +-------------------+
     |     CONFIRMED     |   |     CANCELLED     |
     +-------------------+   +-------------------+
               |                       ^
               v                       | (Only from PENDING/CONFIRMED)
     +-------------------+             |
     |      SHIPPED      |-------------+
     +-------------------+
               |
               v
     +-------------------+
     |     DELIVERED     |
     +-------------------+
```

---

## 4. API Endpoints

### 4.1 Authentication & Health
- `POST /api/auth/login`: Generates signed JWT token with user claims.
- `GET /api/health`: Health status reporting status of PostgreSQL and Redis connections.

### 4.2 Customer Endpoints
- `POST /api/customers`: Register new customer profile (Public).
- `GET /api/customers`: List customers with pagination (Authenticated).
- `GET /api/customers/{id}`: Get customer profile by ID (Authenticated, Redis-cached).
- `PUT /api/customers/{id}`: Update customer profile (Authenticated, cache invalidated).

### 4.3 Order Endpoints
- `POST /api/orders`: Place new order with validated items and calculated total (Authenticated).
- `GET /api/orders`: List orders filtered by `customer_id` and `status` (Authenticated).
- `GET /api/orders/{id}`: Get detailed order with line items (Authenticated, Redis-cached).
- `PATCH /api/orders/{id}/status`: Transition order status (Authenticated, cache invalidated).
- `DELETE /api/orders/{id}`: Cancel order (Authenticated, only valid for `PENDING`/`CONFIRMED`).

---

## 5. Non-Functional Requirements

1. **Security**:
   - Industry-standard JWT (HMAC-SHA256) validation.
   - Protected routes enforce `Authorization: Bearer <token>`.
2. **Performance & Caching**:
   - Redis cache-aside pattern on `GET /api/customers/{id}` and `GET /api/orders/{id}` with 5-minute TTL.
   - Immediate cache eviction on mutation (`PUT`, `PATCH`, `DELETE`).
3. **Resilience & Rate Limiting**:
   - IP-based / Token-based rate limiting (100 requests / minute) to prevent API abuse.
   - Database connection pooling with health monitoring.
4. **Structured Output & Logging**:
   - Gin structured JSON logger including latency, status, path, client IP, and request timestamp.
   - Consistent error envelope: `{"error": "message"}`.
