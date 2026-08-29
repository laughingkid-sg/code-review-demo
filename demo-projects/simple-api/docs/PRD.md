# Product Requirements Document (PRD) — Product Catalog API

## 1. Overview

The Product Catalog API is a lightweight, high-performance RESTful microservice responsible for managing product information within an e-commerce ecosystem. It provides administrative and integration endpoints to create, query, update, and delete product listings with support for filtering, search, and pagination.

## 2. Target Audience & Personas

- **Store Administrator**: Manages product listings, updates inventory numbers, adjusts pricing, and organizes categories.
- **Client Applications (Frontend / Mobile / CI Pipelines)**: Consumes the catalog to display items to shoppers and verify API stability across automated test environments.

## 3. Goals & Non-Goals

### Goals
- Provide standard CRUD operations on product entities.
- Support keyword search by product name/description and category filtering.
- Provide cursor-less offset/limit pagination for listing products.
- Secure administrative/write endpoints using an API key mechanism (`X-API-Key`).
- Maintain zero external service dependencies by utilizing an embedded SQLite database.
- Offer deterministic and reliable execution suitable as a benchmark reference for automated code review tools.

### Non-Goals
- Complex order processing, payment gateway integration, or shopping cart management (handled by downstream services).
- Multi-currency conversions or tax calculations.
- Multi-tenant vendor isolation (reserved for the Complex Marketplace API).

---

## 4. Functional Requirements

### 4.1 Product Entity Attributes
| Field | Type | Description | Constraints |
|---|---|---|---|
| `id` | Integer / String | Unique product identifier (Auto-increment integer or UUID) | Unique, Primary Key |
| `sku` | String | Stock Keeping Unit | Unique, Required, Alphanumeric/Hyphen, 3-50 chars |
| `name` | String | Display name of the product | Required, 1-200 chars |
| `description` | String | Detailed product description | Optional, max 2000 chars |
| `price` | Float / Decimal | Unit price in USD cents or floating point | Required, >= 0.00 |
| `stock` | Integer | Available inventory count | Required, >= 0 |
| `category` | String | Product category name | Required, 1-100 chars |
| `created_at` | Timestamp (ISO 8601) | Record creation timestamp | System generated (UTC) |
| `updated_at` | Timestamp (ISO 8601) | Last modification timestamp | System generated (UTC) |

### 4.2 API Endpoints

#### 1. `GET /api/health`
- **Description**: Returns system health status and uptime indicator.
- **Auth**: Public (No API key required).
- **Response**: `200 OK` with `{"status": "ok", "timestamp": "<ISO-8601>"}`.

#### 2. `GET /api/products`
- **Description**: Lists products matching optional query parameters.
- **Query Parameters**:
  - `page` (int, default `1`): Current page number (1-indexed).
  - `limit` (int, default `10`, max `100`): Number of items per page.
  - `category` (string, optional): Filter by exact category match.
  - `q` (string, optional): Case-insensitive keyword search in name and description.
- **Auth**: Public or Protected (Requires `X-API-Key`).
- **Response**: `200 OK` with JSON array of products and pagination metadata:
  ```json
  {
    "data": [ ... ],
    "pagination": {
      "page": 1,
      "limit": 10,
      "total_items": 42,
      "total_pages": 5
    }
  }
  ```

#### 3. `GET /api/products/{id}`
- **Description**: Retrieves a single product by its ID.
- **Auth**: Public or Protected.
- **Response**: `200 OK` with product payload, or `404 Not Found`.

#### 4. `POST /api/products`
- **Description**: Creates a new product.
- **Auth**: Protected (`X-API-Key` required).
- **Payload**:
  ```json
  {
    "sku": "PROD-1001",
    "name": "Wireless Mechanical Keyboard",
    "description": "75% compact wireless keyboard with hot-swappable switches",
    "price": 89.99,
    "stock": 150,
    "category": "Electronics"
  }
  ```
- **Response**: `201 Created` with created product payload and `Location` header.
- **Errors**: `400 Bad Request` (validation failed), `409 Conflict` (duplicate SKU), `401 Unauthorized`.

#### 5. `PUT /api/products/{id}`
- **Description**: Updates an existing product completely.
- **Auth**: Protected (`X-API-Key` required).
- **Payload**: Full product representation.
- **Response**: `200 OK` with updated product payload.
- **Errors**: `400 Bad Request`, `404 Not Found`, `409 Conflict`, `401 Unauthorized`.

#### 6. `DELETE /api/products/{id}`
- **Description**: Deletes a product by ID.
- **Auth**: Protected (`X-API-Key` required).
- **Response**: `204 No Content`.
- **Errors**: `404 Not Found`, `401 Unauthorized`.

---

## 5. Non-Functional Requirements

### 5.1 Security & Authentication
- Protected endpoints must require an `X-API-Key` header matching the server's configured secret.
- Requests without a valid key must receive `401 Unauthorized` with `{"error": "unauthorized: invalid or missing API key"}`.

### 5.2 Performance & Reliability
- Latency: Sub-15ms p95 response time for read operations under standard test loads.
- Persistence: Data must persist in a local SQLite database file across server restarts.
- Graceful Shutdown: The HTTP server must intercept SIGINT/SIGTERM to drain active connections before terminating.

### 5.3 Error Handling Format
All client and server errors must return a consistent JSON response body:
```json
{
  "error": "human readable error message"
}
```
