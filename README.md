# Code Review Demo

A collection of 3 Go backend API projects at increasing levels of complexity, designed as **runtime code references** for a code review CI/CD pipeline.

## Purpose

These projects are intentionally clean, production-style Go services. They exist so that a separate code review pipeline can be tested and validated against real, working codebases.

## Projects

| Project | Complexity | Description | Stack |
|---------|-----------|-------------|-------|
| `demo-projects/simple-api/` | Beginner | Product Catalog CRUD API | Go stdlib (`net/http`), SQLite |
| `demo-projects/medium-api/` | Intermediate | Order Management API | Gin, PostgreSQL, Redis |
| `demo-projects/complex-api/` | Advanced | Marketplace Platform API | Gin, PostgreSQL, Redis, NATS |

## Each Project Includes

- **`docs/PRD.md`** — Product Requirements Document
- **`docs/TDD.md`** — Tech Design Document
- **`README.md`** — Quick start and API overview
- **`Dockerfile`** + **`docker-compose.yml`** — Docker-runnable
- **Unit tests** — Reasonable coverage
- **`.env.example`** — Environment variable reference

## Quick Start

```bash
# Simple API (port 8080)
cd demo-projects/simple-api && docker compose up -d

# Medium API (port 8081)
cd demo-projects/medium-api && docker compose up -d

# Complex API (port 8082)
cd demo-projects/complex-api && docker compose up -d
```

## Documentation

- [Implementation Plan](docs/IMPLEMENTATION_PLAN.md) — Full technical plan and commit strategy
- [Handover Instructions](docs/HANDOVER.md) — How to build and extend these projects

## Tech Stack

- **Go 1.23**
- **Frameworks**: `net/http` (simple), Gin (medium/complex)
- **Databases**: SQLite (simple), PostgreSQL (medium/complex)
- **Cache**: Redis (medium/complex)
- **Messaging**: NATS (complex)
- **Containerization**: Docker + Docker Compose
