# Product Catalog API (Simple API)

A clean, production-style Go REST API for managing a product catalog. Built with the Go standard library (`net/http`) and an embedded pure-Go SQLite driver (`modernc.org/sqlite`).

## Features

- **Standard Library Routing**: Uses Go 1.22+ enhanced `http.ServeMux` with HTTP method and path parameter pattern matching (`GET /api/products/{id}`).
- **Embedded Database**: Pure Go SQLite storage with WAL (Write-Ahead Logging) mode and connection pooling. No CGo or external database service required.
- **Product Catalog Management**: Full CRUD operations with SKU uniqueness validation, category filtering, search (`q`), and pagination.
- **API Key Security**: Simple and secure API key authentication middleware (`X-API-Key`) for mutation endpoints.
- **Graceful Shutdown**: Intercepts OS signals (`SIGINT`, `SIGTERM`) to cleanly terminate active requests.
- **Docker Ready**: Multi-stage lightweight Alpine container with Docker Compose setup.

---

## API Reference

| Method | Endpoint | Description | Auth Required |
|---|---|---|---|
| `GET` | `/api/health` | Service health status and DB check | No |
| `GET` | `/api/products` | List products (with `page`, `limit`, `category`, `q`) | No |
| `GET` | `/api/products/{id}` | Get product details by ID | No |
| `POST` | `/api/products` | Create a new product | Yes (`X-API-Key`) |
| `PUT` | `/api/products/{id}` | Update an existing product | Yes (`X-API-Key`) |
| `DELETE` | `/api/products/{id}` | Delete a product by ID | Yes (`X-API-Key`) |

---

## Quick Start

### 1. Run Locally

```bash
# Set environment variables (or use defaults)
export PORT=8080
export DB_PATH=data/catalog.db
export API_KEY=dev-api-key-12345

# Run the server
go run cmd/server/main.go
```

### 2. Run with Docker Compose

```bash
docker compose up -d
```

### 3. Run Tests

```bash
go test ./... -v -count=1
```

---

## Example Usage

### Health Check
```bash
curl http://localhost:8080/api/health
```

### Create Product
```bash
curl -X POST http://localhost:8080/api/products \
  -H "Content-Type: application/json" \
  -H "X-API-Key: dev-api-key-12345" \
  -d '{
    "sku": "KB-PRO-01",
    "name": "Mechanical Keyboard",
    "description": "RGB hot-swappable mechanical keyboard",
    "price": 89.99,
    "stock": 45,
    "category": "Keyboards"
  }'
```

### List Products with Filtering & Pagination
```bash
curl "http://localhost:8080/api/products?category=Keyboards&page=1&limit=10"
```

### Get Product
```bash
curl http://localhost:8080/api/products/1
```

### Update Product
```bash
curl -X PUT http://localhost:8080/api/products/1 \
  -H "Content-Type: application/json" \
  -H "X-API-Key: dev-api-key-12345" \
  -d '{
    "sku": "KB-PRO-01",
    "name": "Mechanical Keyboard V2",
    "description": "Upgraded wireless RGB mechanical keyboard",
    "price": 99.99,
    "stock": 50,
    "category": "Keyboards"
  }'
```

### Delete Product
```bash
curl -X DELETE http://localhost:8080/api/products/1 \
  -H "X-API-Key: dev-api-key-12345"
```
