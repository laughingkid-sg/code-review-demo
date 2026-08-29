# Implementation Plan — Three Go Backend API Projects

## Goal

Build 3 production-style Go backend API services at increasing complexity levels under an e-commerce theme. Each project includes a PRD, TDD, unit tests, Dockerfiles, and Docker Compose configurations. These serve as **runtime code references** for a code review CI/CD pipeline.

---

## Project Overview

| Aspect | Simple | Medium | Complex |
|--------|--------|--------|---------|
| **Project** | Product Catalog API | Order Management API | Marketplace Platform API |
| **Directory** | `demo-projects/simple-api/` | `demo-projects/medium-api/` | `demo-projects/complex-api/` |
| **Framework** | `net/http` (stdlib) | Gin | Gin |
| **Database** | SQLite (embedded, pure Go) | PostgreSQL | PostgreSQL |
| **Cache** | None | Redis | Redis |
| **Message Queue** | None | None | NATS |
| **Auth** | API key (header) | JWT | JWT + RBAC (roles) |
| **Architecture** | Flat package | Layered (handler→service→repo) | Clean Architecture + Domain Events |
| **Docker** | Single Dockerfile | Dockerfile + docker-compose (PG, Redis) | Dockerfile + docker-compose (PG, Redis, NATS) |
| **Tests** | Handler-level tests | Unit + integration tests | Unit + integration + domain tests |
| **Port** | `:8080` | `:8081` | `:8082` |
| **Go module** | `github.com/demo/simple-api` | `github.com/demo/medium-api` | `github.com/demo/complex-api` |

---

## 1. Simple API — Product Catalog

A straightforward CRUD REST API for managing a product catalog. Uses Go standard library (`net/http` with Go 1.22+ routing patterns) and SQLite (`modernc.org/sqlite`, pure Go — no CGo).

### Directory Structure

```
simple-api/
├── docs/
│   ├── PRD.md
│   ├── TDD.md
│   ├── API.md
│   └── postman_collection.json
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── handler/
│   │   ├── product.go
│   │   ├── product_test.go
│   │   └── health.go
│   ├── model/
│   │   └── product.go
│   ├── store/
│   │   ├── sqlite.go
│   │   └── sqlite_test.go
│   └── middleware/
│       └── apikey.go
├── migrations/
│   └── 001_create_products.sql
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.yml
├── .env.example
└── README.md
```

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/health` | Health check |
| `GET` | `/api/products` | List products (pagination, search) |
| `GET` | `/api/products/{id}` | Get product by ID |
| `POST` | `/api/products` | Create product |
| `PUT` | `/api/products/{id}` | Update product |
| `DELETE` | `/api/products/{id}` | Delete product |

---

## 2. Medium API — Order Management

A multi-layered REST API for managing customers and orders. Uses Gin, PostgreSQL, and Redis.

### Directory Structure

```
medium-api/
├── docs/
│   ├── PRD.md
│   ├── TDD.md
│   ├── API.md
│   └── postman_collection.json
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── handler/
│   │   ├── customer.go
│   │   ├── customer_test.go
│   │   ├── order.go
│   │   ├── order_test.go
│   │   └── health.go
│   ├── service/
│   │   ├── customer.go
│   │   ├── customer_test.go
│   │   ├── order.go
│   │   └── order_test.go
│   ├── repository/
│   │   ├── customer_pg.go
│   │   ├── customer_pg_test.go
│   │   ├── order_pg.go
│   │   └── order_pg_test.go
│   ├── model/
│   │   ├── customer.go
│   │   ├── order.go
│   │   └── errors.go
│   ├── middleware/
│   │   ├── auth.go
│   │   ├── logger.go
│   │   └── ratelimit.go
│   └── cache/
│       ├── redis.go
│       └── redis_test.go
├── migrations/
│   ├── 001_create_customers.sql
│   └── 002_create_orders.sql
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.yml
├── .env.example
└── README.md
```

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/auth/login` | Authenticate, get JWT |
| `GET` | `/api/customers` | List customers |
| `POST` | `/api/customers` | Create customer |
| `GET` | `/api/customers/{id}` | Get customer |
| `PUT` | `/api/customers/{id}` | Update customer |
| `GET` | `/api/orders` | List orders (filter by status, customer) |
| `POST` | `/api/orders` | Create order |
| `GET` | `/api/orders/{id}` | Get order with items |
| `PATCH` | `/api/orders/{id}/status` | Update order status |
| `DELETE` | `/api/orders/{id}` | Cancel order |

### Architecture

```
HTTP Handler Layer (Gin) → Service Layer (Business Logic) → Repository Layer (PostgreSQL)
                                    ↓
                              Cache Layer (Redis)
```

---

## 3. Complex API — Marketplace Platform

A full-featured marketplace platform with multi-tenant vendor support, async event processing (NATS), and clean architecture.

### Directory Structure

```
complex-api/
├── docs/
│   ├── PRD.md
│   ├── TDD.md
│   ├── API.md
│   └── postman_collection.json
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── domain/
│   │   ├── vendor/
│   │   │   ├── entity.go
│   │   │   ├── repository.go
│   │   │   ├── service.go
│   │   │   └── service_test.go
│   │   ├── product/
│   │   │   ├── entity.go
│   │   │   ├── repository.go
│   │   │   ├── service.go
│   │   │   └── service_test.go
│   │   ├── order/
│   │   │   ├── entity.go
│   │   │   ├── repository.go
│   │   │   ├── service.go
│   │   │   └── service_test.go
│   │   └── event/
│   │       ├── event.go
│   │       └── publisher.go
│   ├── adapter/
│   │   ├── http/
│   │   │   ├── router.go
│   │   │   ├── vendor_handler.go
│   │   │   ├── vendor_handler_test.go
│   │   │   ├── product_handler.go
│   │   │   ├── product_handler_test.go
│   │   │   ├── order_handler.go
│   │   │   ├── order_handler_test.go
│   │   │   └── middleware/
│   │   │       ├── auth.go
│   │   │       ├── logger.go
│   │   │       └── cors.go
│   │   ├── postgres/
│   │   │   ├── vendor_repo.go
│   │   │   ├── vendor_repo_test.go
│   │   │   ├── product_repo.go
│   │   │   ├── product_repo_test.go
│   │   │   ├── order_repo.go
│   │   │   └── order_repo_test.go
│   │   ├── redis/
│   │   │   ├── cache.go
│   │   │   └── cache_test.go
│   │   └── nats/
│   │       ├── publisher.go
│   │       ├── subscriber.go
│   │       └── subscriber_test.go
│   ├── usecase/
│   │   ├── vendor_usecase.go
│   │   ├── product_usecase.go
│   │   ├── order_usecase.go
│   │   └── order_usecase_test.go
│   └── pkg/
│       ├── pagination/
│       │   └── pagination.go
│       ├── validator/
│       │   └── validator.go
│       └── response/
│           └── response.go
├── migrations/
│   ├── 001_create_vendors.sql
│   ├── 002_create_products.sql
│   ├── 003_create_orders.sql
│   └── 004_create_events.sql
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.yml
├── .env.example
└── README.md
```

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/auth/login` | Authenticate (vendor/admin/buyer) |
| `GET` | `/api/vendors` | List vendors |
| `POST` | `/api/vendors` | Register vendor (admin) |
| `GET` | `/api/vendors/{id}` | Get vendor profile |
| `PUT` | `/api/vendors/{id}` | Update vendor |
| `GET` | `/api/vendors/{id}/products` | Vendor's products |
| `GET` | `/api/products` | Marketplace listing (search, filter, paginate) |
| `POST` | `/api/products` | Create product (vendor) |
| `GET` | `/api/products/{id}` | Product detail |
| `PUT` | `/api/products/{id}` | Update product (vendor/owner) |
| `POST` | `/api/orders` | Place order (buyer) |
| `GET` | `/api/orders` | List orders (filtered by role) |
| `GET` | `/api/orders/{id}` | Order detail |
| `PATCH` | `/api/orders/{id}/status` | Update order status |

### RBAC Roles

| Role | Capabilities |
|------|-------------|
| `admin` | Full access, manage vendors |
| `vendor` | CRUD own products, view own orders |
| `buyer` | Browse products, place/view own orders |

### Architecture

```
Adapter Layer (HTTP Handlers, PostgreSQL Repos, Redis Cache, NATS Publisher/Subscriber)
        ↓
Use Case Layer (Application Use Cases)
        ↓
Domain Layer (Entities, Services, Domain Events)
```

---

## Conventional Commit Strategy

Each change is a small, focused commit following [Conventional Commits](https://www.conventionalcommits.org/).

### Phase 0: Repository Setup
| # | Commit Message |
|---|---------------|
| 1 | `chore: add .gitignore and .gitattributes` |
| 2 | `docs: add repository README` |
| 3 | `docs: add implementation plan` |
| 4 | `docs: add handover instructions` |

### Phase 1: Simple API — Product Catalog
| # | Commit Message |
|---|---------------|
| 5 | `docs(simple-api): add PRD for product catalog API` |
| 6 | `docs(simple-api): add tech design document` |
| 7 | `feat(simple-api): initialize Go module` |
| 8 | `feat(simple-api): add product model and validation` |
| 9 | `feat(simple-api): add SQLite store implementation` |
| 10 | `feat(simple-api): add database migration` |
| 11 | `feat(simple-api): add API key auth middleware` |
| 12 | `feat(simple-api): add health check handler` |
| 13 | `feat(simple-api): add product CRUD handlers` |
| 14 | `test(simple-api): add store unit tests` |
| 15 | `test(simple-api): add handler unit tests` |
| 16 | `feat(simple-api): add server entry point` |
| 17 | `feat(simple-api): add Dockerfile and docker-compose` |
| 18 | `docs(simple-api): add README and .env.example` |

### Phase 2: Medium API — Order Management
| # | Commit Message |
|---|---------------|
| 19 | `docs(medium-api): add PRD for order management API` |
| 20 | `docs(medium-api): add tech design document` |
| 21 | `feat(medium-api): initialize Go module` |
| 22 | `feat(medium-api): add config management` |
| 23 | `feat(medium-api): add customer and order models` |
| 24 | `feat(medium-api): add custom error types` |
| 25 | `feat(medium-api): add customer PostgreSQL repository` |
| 26 | `feat(medium-api): add order PostgreSQL repository` |
| 27 | `feat(medium-api): add Redis cache layer` |
| 28 | `feat(medium-api): add customer service with business logic` |
| 29 | `feat(medium-api): add order service with business logic` |
| 30 | `feat(medium-api): add JWT auth middleware` |
| 31 | `feat(medium-api): add logging and rate limit middleware` |
| 32 | `feat(medium-api): add health check handler` |
| 33 | `feat(medium-api): add customer handlers` |
| 34 | `feat(medium-api): add order handlers` |
| 35 | `feat(medium-api): add database migrations` |
| 36 | `test(medium-api): add repository unit tests` |
| 37 | `test(medium-api): add service unit tests` |
| 38 | `test(medium-api): add handler unit tests` |
| 39 | `test(medium-api): add cache unit tests` |
| 40 | `feat(medium-api): add server entry point` |
| 41 | `feat(medium-api): add Dockerfile and docker-compose` |
| 42 | `docs(medium-api): add README, API docs, Postman collection and .env.example` |

### Phase 3: Complex API — Marketplace Platform
| # | Commit Message |
|---|---------------|
| 43 | `docs(complex-api): add PRD for marketplace platform API` |
| 44 | `docs(complex-api): add tech design document` |
| 45 | `feat(complex-api): initialize Go module` |
| 46 | `feat(complex-api): add config management` |
| 47 | `feat(complex-api): add domain event types and publisher interface` |
| 48 | `feat(complex-api): add vendor domain entity and repository interface` |
| 49 | `feat(complex-api): add product domain entity and repository interface` |
| 50 | `feat(complex-api): add order domain entity and repository interface` |
| 51 | `feat(complex-api): add vendor domain service` |
| 52 | `feat(complex-api): add product domain service` |
| 53 | `feat(complex-api): add order domain service` |
| 54 | `feat(complex-api): add PostgreSQL vendor repository` |
| 55 | `feat(complex-api): add PostgreSQL product repository` |
| 56 | `feat(complex-api): add PostgreSQL order repository` |
| 57 | `feat(complex-api): add Redis cache adapter` |
| 58 | `feat(complex-api): add NATS event publisher` |
| 59 | `feat(complex-api): add NATS event subscriber` |
| 60 | `feat(complex-api): add shared packages (pagination, validator, response)` |
| 61 | `feat(complex-api): add application use cases` |
| 62 | `feat(complex-api): add HTTP middleware (auth, logger, CORS)` |
| 63 | `feat(complex-api): add vendor HTTP handler` |
| 64 | `feat(complex-api): add product HTTP handler` |
| 65 | `feat(complex-api): add order HTTP handler` |
| 66 | `feat(complex-api): add HTTP router setup` |
| 67 | `feat(complex-api): add database migrations` |
| 68 | `test(complex-api): add domain service tests` |
| 69 | `test(complex-api): add repository tests` |
| 70 | `test(complex-api): add handler tests` |
| 71 | `test(complex-api): add use case tests` |
| 72 | `test(complex-api): add NATS subscriber tests` |
| 73 | `test(complex-api): add cache tests` |
| 74 | `feat(complex-api): add server entry point` |
| 75 | `feat(complex-api): add Dockerfile and docker-compose` |
| 76 | `docs(complex-api): add README, API docs, Postman collection and .env.example` |

---

## Verification Plan

### Automated Tests

```bash
cd demo-projects/simple-api && go test ./... -v -count=1
cd demo-projects/medium-api && go test ./... -v -count=1
cd demo-projects/complex-api && go test ./... -v -count=1
```

### Build Verification

```bash
cd demo-projects/simple-api && go build ./... && go vet ./...
cd demo-projects/medium-api && go build ./... && go vet ./...
cd demo-projects/complex-api && go build ./... && go vet ./...
```

### Docker Verification

```bash
cd demo-projects/simple-api && docker compose up -d && curl http://localhost:8080/api/health
cd demo-projects/medium-api && docker compose up -d && curl http://localhost:8081/api/health
cd demo-projects/complex-api && docker compose up -d && curl http://localhost:8082/api/health
```
