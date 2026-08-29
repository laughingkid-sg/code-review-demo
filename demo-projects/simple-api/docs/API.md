# Product Catalog API Specification & Documentation

- **Service Name**: Product Catalog API (`simple-api`)
- **Version**: `1.0.0`
- **Default Base URL**: `http://localhost:8080`
- **Content Type**: `application/json`

---

## 1. Authentication

The API uses **Header-based API Key authentication** for all state-modifying operations (`POST`, `PUT`, `DELETE`).

| Header | Description | Default Development Key |
|---|---|---|
| `X-API-Key` | Secret authentication token | `dev-api-key-12345` |

If the key is missing or invalid on protected endpoints, the server responds with:
```http
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{
  "error": "unauthorized: invalid or missing API key"
}
```

---

## 2. Standard Error Format

All error responses return a standardized JSON structure with appropriate HTTP status codes:

```json
{
  "error": "Human-readable description of what went wrong"
}
```

### Common HTTP Status Codes:
- `200 OK`: Request succeeded.
- `201 Created`: Resource successfully created (`Location` header included).
- `204 No Content`: Resource successfully deleted.
- `400 Bad Request`: Validation failure or invalid JSON payload.
- `401 Unauthorized`: Missing or invalid `X-API-Key`.
- `404 Not Found`: Requested resource does not exist.
- `409 Conflict`: Unique constraint violation (e.g. duplicate SKU).
- `500 Internal Server Error`: Server-side processing failure.
- `503 Service Unavailable`: Dependent service/database unavailable.

---

## 3. Endpoints

### 3.1 Health Check

Checks system health and database connectivity.

- **Method**: `GET`
- **Path**: `/api/health`
- **Authentication**: None (Public)

#### Request Example
```bash
curl http://localhost:8080/api/health
```

#### Response Example (`200 OK`)
```json
{
  "status": "ok",
  "timestamp": "2026-08-29T03:08:52Z",
  "database": "connected"
}
```

---

### 3.2 List Products

Retrieves a paginated list of products with optional search and category filters.

- **Method**: `GET`
- **Path**: `/api/products`
- **Authentication**: None (Public)

#### Query Parameters

| Parameter | Type | Default | Description |
|---|---|---|---|
| `page` | integer | `1` | Page number (1-indexed) |
| `limit` | integer | `10` | Number of items per page (min: 1, max: 100) |
| `category` | string | `""` | Filter by exact category match |
| `q` | string | `""` | Search query matching product `name` or `description` |

#### Request Example
```bash
curl "http://localhost:8080/api/products?page=1&limit=10&category=Keyboards&q=RGB"
```

#### Response Example (`200 OK`)
```json
{
  "data": [
    {
      "id": 1,
      "sku": "KB-PRO-01",
      "name": "Mechanical Keyboard",
      "description": "RGB hot-swappable mechanical keyboard",
      "price": 89.99,
      "stock": 45,
      "category": "Keyboards",
      "created_at": "2026-08-29T03:00:00Z",
      "updated_at": "2026-08-29T03:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 10,
    "total_items": 1,
    "total_pages": 1
  }
}
```

---

### 3.3 Get Product by ID

Retrieves details of a single product.

- **Method**: `GET`
- **Path**: `/api/products/{id}`
- **Authentication**: None (Public)

#### Path Parameters
- `id` (integer, required): The unique numeric product ID.

#### Request Example
```bash
curl http://localhost:8080/api/products/1
```

#### Response Example (`200 OK`)
```json
{
  "id": 1,
  "sku": "KB-PRO-01",
  "name": "Mechanical Keyboard",
  "description": "RGB hot-swappable mechanical keyboard",
  "price": 89.99,
  "stock": 45,
  "category": "Keyboards",
  "created_at": "2026-08-29T03:00:00Z",
  "updated_at": "2026-08-29T03:00:00Z"
}
```

#### Error Example (`404 Not Found`)
```json
{
  "error": "product not found"
}
```

---

### 3.4 Create Product

Creates a new product record.

- **Method**: `POST`
- **Path**: `/api/products`
- **Authentication**: Required (`X-API-Key`)

#### Request Headers
- `Content-Type: application/json`
- `X-API-Key: dev-api-key-12345`

#### Request Body Schema

| Field | Type | Required | Constraints | Description |
|---|---|---|---|---|
| `sku` | string | Yes | `3..50` chars, `[A-Za-z0-9_-]` | Unique stock keeping unit |
| `name` | string | Yes | `1..200` chars | Product title |
| `description` | string | No | `0..2000` chars | Detailed product description |
| `price` | number | Yes | `>= 0.00` | Price in USD |
| `stock` | integer | Yes | `>= 0` | Current inventory count |
| `category` | string | Yes | `1..100` chars | Product categorization |

#### Request Example
```bash
curl -X POST http://localhost:8080/api/products \
  -H "Content-Type: application/json" \
  -H "X-API-Key: dev-api-key-12345" \
  -d '{
    "sku": "MOU-PRO-01",
    "name": "Ultra-light Wireless Mouse",
    "description": "Ergonomic 26K DPI wireless gaming mouse",
    "price": 69.99,
    "stock": 100,
    "category": "Accessories"
  }'
```

#### Response Example (`201 Created`)
- **Header**: `Location: /api/products/2`
```json
{
  "id": 2,
  "sku": "MOU-PRO-01",
  "name": "Ultra-light Wireless Mouse",
  "description": "Ergonomic 26K DPI wireless gaming mouse",
  "price": 69.99,
  "stock": 100,
  "category": "Accessories",
  "created_at": "2026-08-29T03:10:00Z",
  "updated_at": "2026-08-29T03:10:00Z"
}
```

#### Error Example (`409 Conflict`)
```json
{
  "error": "product with this SKU already exists"
}
```

---

### 3.5 Update Product

Updates all fields of an existing product.

- **Method**: `PUT`
- **Path**: `/api/products/{id}`
- **Authentication**: Required (`X-API-Key`)

#### Path Parameters
- `id` (integer, required): The product ID to update.

#### Request Example
```bash
curl -X PUT http://localhost:8080/api/products/1 \
  -H "Content-Type: application/json" \
  -H "X-API-Key: dev-api-key-12345" \
  -d '{
    "sku": "KB-PRO-01",
    "name": "Mechanical Keyboard RGB V2",
    "description": "Upgraded hot-swappable keyboard with dampener foam",
    "price": 94.99,
    "stock": 40,
    "category": "Keyboards"
  }'
```

#### Response Example (`200 OK`)
```json
{
  "id": 1,
  "sku": "KB-PRO-01",
  "name": "Mechanical Keyboard RGB V2",
  "description": "Upgraded hot-swappable keyboard with dampener foam",
  "price": 94.99,
  "stock": 40,
  "category": "Keyboards",
  "created_at": "2026-08-29T03:00:00Z",
  "updated_at": "2026-08-29T03:15:00Z"
}
```

---

### 3.6 Delete Product

Deletes a product by ID.

- **Method**: `DELETE`
- **Path**: `/api/products/{id}`
- **Authentication**: Required (`X-API-Key`)

#### Path Parameters
- `id` (integer, required): The product ID to remove.

#### Request Example
```bash
curl -X DELETE http://localhost:8080/api/products/1 \
  -H "X-API-Key: dev-api-key-12345"
```

#### Response Example (`204 No Content`)
*(Empty response body)*
