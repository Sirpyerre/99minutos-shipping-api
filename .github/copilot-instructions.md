# Copilot Instructions – Shipping Tracking System (99minutos)

Go 1.25 + Echo v4 event-driven logistics API. Processes 10,000+ events/second with guaranteed per-shipment ordering.

## Architecture

Clean architecture with three layers—never let outer layers bleed into inner ones:

```
internal/domain/          ← Business entities & state machine (no external deps)
internal/application/     ← Use cases + DTOs (depends only on domain)
internal/infrastructure/  ← Echo handlers, MongoDB, Redis, event queue (depends on app + domain)
```

Key infrastructure directories:
- `internal/infrastructure/http/handlers/` – Echo HTTP handlers
- `internal/infrastructure/queue/` – Per-shipment Go channel queue + worker pool
- `internal/infrastructure/mongo/` – MongoDB client & repository
- `internal/infrastructure/redis/` – Redis deduplication checker
- `internal/shared/` – Config, logger (slog), JWT helpers, errors

## Code Style

- Use `log/slog` (stdlib) for structured logging: `logger.Info("msg", "key", val)`
- Constructor pattern for dependency injection: `NewXxxUseCase(db, logger)`
- Context propagation: always pass `ctx := c.Request().Context()` from handlers
- Error handling in handlers: log with `logger.Error(...)`, return `application.ErrorResponse{Error: ...}`
- Tracking numbers formatted as `99M-<8-char-uuid-prefix>`

## Build and Test

```bash
# Local dev (recommended)
make docker-up        # Start MongoDB + Redis + API via docker-compose
make test-race        # Run all tests with race detector (required before commit)
make test-coverage    # Coverage report (target: 80%+)
make lint             # golangci-lint
make fmt              # go fmt
make build            # Outputs binary to ./bin/server

# Manual
go mod download && go mod tidy
go run ./cmd/server/main.go
curl http://localhost:8080/health  # → {"status":"ok"}
```

Seeded credentials: `admin_user / password123` (role: admin), `client_user_001 / password123` (role: client, client_id: client_001).

## Project Conventions

**Adding a new endpoint** – follow this three-file pattern:
1. Add request/response DTOs to `internal/application/dto.go`
2. Create use case in `internal/application/<action>_shipment.go`
3. Add/update handler in `internal/infrastructure/http/handlers/`; register route in `router.go`

**State machine** – valid transitions live exclusively in `internal/domain/` (`status.go`). Never duplicate or override them in handlers or use cases.

```go
// validTransitions whitelist (domain/status.go)
Created → PickedUp | Cancelled
PickedUp → InWarehouse | Cancelled
InWarehouse → InTransit | Cancelled
InTransit → Delivered
Delivered → (terminal)
Cancelled → (terminal)
```

**Event processing** – events return `202 Accepted` immediately; processing is async via per-shipment Go channels (`map[tracking_number]chan *Event` with `sync.RWMutex`). Dedup key: `dedup:<tracking_number>:<source>:<unix_timestamp>` in Redis (TTL 1 hour).

**MongoDB schema** – `status_history` is embedded in the shipment document for single-query reads. Separate `status_events` collection is an audit trail only. Collections: `shipments`, `status_events`, `auth_users`.

**RBAC** – JWT claims carry `role` + `client_id`. `client` role: filter by own `client_id`. `admin` role: no filter applied. Enforced in middleware, not in use cases.

## Integration Points

| Service | Purpose | Config env var |
|---------|---------|----------------|
| MongoDB 7 | Primary persistence | `MONGO_URI`, `MONGO_DB` |
| Redis 7 | Deduplication cache | `REDIS_ADDR`, `REDIS_DB` |
| Echo v4 | HTTP routing + middleware | – |
| golang-jwt | Stateless auth | `JWT_SECRET` |

## Security

- JWT secret from `JWT_SECRET` env var — never hardcode
- Passwords stored as bcrypt hashes in `auth_users`
- RBAC enforced by middleware before handlers execute; handlers trust `token.client_id` for filtering
