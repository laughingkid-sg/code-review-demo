# Handover Instructions

## Context

This repository contains 3 Go backend API projects that serve as **runtime code references** for a code review CI/CD pipeline. The projects are intentionally clean, bug-free, and production-style so the review pipeline can be tested against realistic codebases.

**The code review pipeline is being built separately.** This repo only contains the reference projects.

---

## Current Status

- [x] Repository initialized with `.gitignore`, `.gitattributes`, `README.md`
- [x] Implementation plan written (`docs/IMPLEMENTATION_PLAN.md`)
- [x] Handover instructions written (this file)
- [ ] **Phase 1**: Simple API — Product Catalog (`demo-projects/simple-api/`)
- [ ] **Phase 2**: Medium API — Order Management (`demo-projects/medium-api/`)
- [ ] **Phase 3**: Complex API — Marketplace Platform (`demo-projects/complex-api/`)

---

## How to Continue

### Prerequisites

- Go 1.23 installed
- Docker and Docker Compose installed
- Git configured

### Build Order

**Follow the phases in order.** Each phase is independent but builds on the complexity of the previous one.

1. **Read the implementation plan** in `docs/IMPLEMENTATION_PLAN.md` — it contains the full directory structure, endpoints, architecture, and commit strategy for all 3 projects.

2. **Follow the conventional commit sequence** listed in the plan. Each commit should be small, focused, and follow [Conventional Commits](https://www.conventionalcommits.org/) format.

3. **For each project**, build in this order:
   - PRD and TDD docs first (`docs/PRD.md`, `docs/TDD.md`)
   - Initialize Go module (`go mod init`)
   - Models/entities (data layer)
   - Store/repository (persistence layer)
   - Migrations (SQL schemas)
   - Middleware (auth, logging)
   - Handlers (HTTP layer)
   - Tests (unit tests per layer)
   - Entry point (`cmd/server/main.go`)
   - Docker files (`Dockerfile`, `docker-compose.yml`)
   - Project README and `.env.example`

4. **Verify each project** before moving to the next:
   ```bash
   go build ./...
   go vet ./...
   go test ./... -v -count=1
   docker compose build
   docker compose up -d
   curl http://localhost:<port>/api/health
   docker compose down
   ```

---

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| **SQLite for simple project** | Zero-dependency local dev; uses `modernc.org/sqlite` (pure Go, no CGo) |
| **`net/http` for simple, Gin for medium/complex** | Shows progression from stdlib to framework |
| **Separate Go modules per project** | Each project is independently buildable and testable |
| **No intentional bugs** | These are reference codebases; bugs will be introduced separately by the review pipeline |
| **Docker Compose for all** | Consistent `docker compose up` experience across all projects |
| **Conventional commits** | Clean git history for the review pipeline to analyze |

---

## Port Allocation

| Project | Port |
|---------|------|
| Simple API | `:8080` |
| Medium API | `:8081` |
| Complex API | `:8082` |

---

## Important Notes

- **Do NOT introduce bugs** — these projects must compile, pass tests, and run in Docker cleanly.
- **Each commit should be atomic** — one logical change per commit.
- **Run `go vet` and tests before each commit** to ensure nothing is broken.
- **All three projects must be independently runnable** via `docker compose up`.
