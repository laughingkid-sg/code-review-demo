# Order Management API (Medium API)

A layered Go REST API for managing customers and orders. The service uses Gin, PostgreSQL, Redis, and JWT auth, and is packaged to run locally or with Docker Compose.

## Features

- Customer registration and profile management
- JWT-protected customer and order endpoints
- Order lifecycle state transitions
- Redis cache-aside reads for customer and order detail endpoints
- PostgreSQL persistence with SQL migrations
- Docker Compose orchestration for the API, Postgres, and Redis

## Project Layout

- `cmd/server/main.go` - application entrypoint
- `internal/handler` - HTTP layer
- `internal/service` - business logic
- `internal/repository` - PostgreSQL persistence
- `internal/cache` - Redis cache layer
- `migrations` - database schema

## Run with Docker

```bash
docker compose up --build
```

Then check health:

```bash
curl http://localhost:8081/api/health
```

## Local Development

Copy the environment template and adjust values if needed:

```bash
cp .env.example .env
```

Run the service:

```bash
go run ./cmd/server
```

## Login

For demo purposes, the auth handler accepts:

- `admin@demo.com` / `admin123`

It returns a JWT you can use as `Authorization: Bearer <token>`.

## Verification

```bash
go test ./...
docker compose build
docker compose up -d
curl http://localhost:8081/api/health
docker compose down
```
