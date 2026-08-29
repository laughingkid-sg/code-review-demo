# Order Management API

Base URL: `http://localhost:8081`

## Health

`GET /api/health`

Response:

```json
{
  "status": "ok",
  "timestamp": "2026-08-29T12:00:00Z",
  "database": "connected",
  "cache": "connected"
}
```

## Authentication

`POST /api/auth/login`

Request:

```json
{
  "email": "admin@demo.com",
  "password": "admin123"
}
```

## Customers

`POST /api/customers`

`GET /api/customers`

`GET /api/customers/{id}`

`PUT /api/customers/{id}`

`DELETE /api/customers/{id}`

## Orders

`POST /api/orders`

`GET /api/orders`

`GET /api/orders/{id}`

`PATCH /api/orders/{id}/status`

`DELETE /api/orders/{id}`

## Common Error Shape

```json
{
  "error": "message"
}
```
