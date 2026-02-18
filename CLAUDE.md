# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Event-driven shipment tracking system for 99minutos (Latin American logistics). Go 1.25 + Echo v4, MongoDB 7, Redis 7. Targets 10,000+ events/second with guaranteed per-shipment ordering.

## Build & Run Commands

```bash
# Start infrastructure (MongoDB + Redis + API)
make docker-up            # or: docker-compose up -d

# Build
make build                # outputs ./bin/server
go run ./cmd/server/main.go

# Test
go test ./... -v          # all tests
go test ./tests/unit/... -v   # unit only
go test ./... -race       # race detector (required before commit)
go test ./... -cover      # coverage (target: 80%+)
make test-race            # shortcut
make test-coverage        # HTML coverage report

# Run a single test
go test ./tests/unit/... -run TestValidTransitions -v

# Lint & format
make lint                 # golangci-lint
make fmt                  # go fmt + goimports

# Health check
curl http://localhost:8080/health   # → {"status":"ok"}
```

## Architecture (Clean Architecture, 3 layers)

```
internal/domain/          ← Business entities & state machine (zero external deps)
internal/application/     ← Use cases + DTOs (depends only on domain)
internal/infrastructure/  ← HTTP handlers, MongoDB, Redis, event queue
```

- **domain/**: `shipment.go` (entity), `status.go` (state machine with valid transitions whitelist). This is the source of truth for business rules.
- **application/**: Use cases (`create_shipment.go`, `get_shipment.go`, `list_shipments.go`, `process_event.go`) + `dto.go` (all request/response structs).
- **infrastructure/http/handlers/**: Echo HTTP handlers. Routes registered in `router.go`.
- **infrastructure/queue/**: Per-shipment Go channel queue (`map[tracking_number]chan *Event` with `sync.RWMutex`) + worker pool for async event processing.
- **infrastructure/mongo/**: MongoDB client, repository, migrations.
- **infrastructure/redis/**: Dedup checker (key: `dedup:<tracking>:<source>:<unix_ts>`, TTL 1hr).

Entry point: `cmd/server/main.go`

## Key Conventions

**Adding a new endpoint** — always follow this 3-file pattern:
1. DTOs in `internal/application/dto.go`
2. Use case in `internal/application/<action>.go`
3. Handler in `internal/infrastructure/http/handlers/`; register route in `router.go`

**Dependency injection**: Constructor pattern `NewXxxUseCase(db *mongo.Database, logger *slog.Logger)`.

**Context propagation**: Always `ctx := c.Request().Context()` from handlers into use cases.

**Logging**: Use `log/slog` (stdlib). `logger.Info("msg", "key", val)`.

**Event processing**: POST /events returns `202 Accepted` immediately; processing is async via per-shipment channels. Dedup in Redis, then validate state transition, then persist to MongoDB atomically.

**State machine** (in `domain/status.go`):
```
created → picked_up | cancelled
picked_up → in_warehouse | cancelled
in_warehouse → in_transit | cancelled
in_transit → delivered
delivered → (terminal)
cancelled → (terminal)
```

**RBAC**: JWT claims carry `role` + `client_id`. Middleware enforces filtering — `client` sees own shipments only, `admin` sees all. Handlers trust middleware-extracted claims.

**MongoDB collections**: `shipments` (with embedded `status_history` array), `status_events` (audit trail), `auth_users`.

**Tracking numbers**: Format `99M-<8-char-uuid-prefix>`.

## Test Credentials

- Admin: `admin_user` / `password123` (role: admin)
- Client: `client_user_001` / `password123` (role: client, client_id: client_001)

## Environment Variables

`PORT`, `ENV`, `MONGO_URI`, `MONGO_DB`, `REDIS_ADDR`, `REDIS_DB`, `JWT_SECRET`, `LOG_LEVEL`. See `.env.example`.
