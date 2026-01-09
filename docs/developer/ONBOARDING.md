# Developer Onboarding Guide

Welcome to Open Model Gateway! This guide will help you get up to speed with the codebase.

## Prerequisites

- Go 1.21+
- Node.js 18+ or Bun
- Docker (for local Postgres/Redis)
- Make

## Quick Start

```bash
# Clone and enter the repo
git clone <repo-url>
cd open_model_gateway

# Start infrastructure
make compose-up

# Run the backend (builds UI automatically)
make run-backend

# In another terminal, run frontend dev server (optional, for HMR)
cd backend/frontend && bun run dev
```

## Project Structure

```
open_model_gateway/
├── backend/                 # Go backend
│   ├── cmd/routerd/        # Main entry point
│   ├── internal/           # Private packages
│   │   ├── app/            # Application container and context
│   │   ├── adapters/       # Provider adapters (OpenAI, Anthropic, etc.)
│   │   ├── config/         # Configuration loading
│   │   ├── db/             # SQLC-generated database code
│   │   ├── httpserver/     # HTTP handlers (admin, user, public)
│   │   ├── router/         # Request routing engine
│   │   ├── runtime/        # Application bootstrap
│   │   └── services/       # Business logic
│   ├── migrations/         # Goose migrations
│   ├── sql/                # SQL schema and queries
│   └── frontend/           # React frontend
├── deploy/                 # Deployment configs
└── docs/                   # Documentation
```

## Key Concepts

### 1. Container Architecture
The `app.Container` holds all dependencies, decomposed into sub-containers:
- `ServiceContainer` - Business services
- `RoutingContainer` - Provider routing
- `DataContainer` - Database and Redis
- `TelemetryContainer` - Observability
- `RateLimitContainer` - Rate limiting

See [ADR 001](../architecture/001-container-decomposition.md).

### 2. Provider Adapters
Each LLM provider has an adapter in `internal/adapters/`:
- Implements common interfaces for chat, embeddings, etc.
- Handles provider-specific request/response translation
- Base utilities in `adapters/base/`

### 3. Request Routing
The routing engine (`internal/router/`) handles:
- Model alias to provider mapping
- Weighted provider selection
- Health-aware failover
- Rate limit enforcement

### 4. Database Layer
- Schema: `sql/schema/*.sql`
- Queries: `sql/queries/*.sql`
- Generated code: `internal/db/` (via `sqlc generate`)
- Migrations: `migrations/` (via Goose)

### 5. Frontend
React/Vite app with two portals:
- Admin portal (`/admin/`) - Tenant management
- User portal (`/`) - Self-service

## Common Tasks

### Adding a New Provider

1. Create adapter in `internal/adapters/<provider>/`
2. Implement required interfaces (ChatAdapter, etc.)
3. Add builder in `internal/providers/builder_<provider>.go`
4. Register in `internal/providers/factory.go`
5. Add config in `internal/config/config.go`

### Adding a Database Query

1. Add SQL to `sql/queries/<domain>.sql`
2. Run `cd backend && sqlc generate`
3. Use generated functions in services

### Adding an API Endpoint

1. Add handler in appropriate `httpserver/` subpackage
2. Register route in the routes file
3. Add request/response types
4. Update frontend API client if needed

## Testing

```bash
# Run all backend tests
make test-backend

# Run specific package tests
cd backend && go test ./internal/adapters/...

# Run frontend tests
cd backend/frontend && bun run test
```

### Test Utilities
The `internal/testutil` package provides:
- `StartTestPostgres()` - Embedded Postgres
- `StartMiniRedis()` - Redis mock
- `NewTestConfig()` - Test configuration
- `Fixtures` - Test data helpers

## Configuration

Primary config: `deploy/router.local.yaml`

Key sections:
- `server` - HTTP settings
- `database` - Postgres connection
- `redis` - Redis connection
- `providers` - API keys for each provider
- `model_catalog` - Model aliases and routing
- `rate_limits` / `budgets` - Default limits

Environment overrides use `ROUTER_` prefix.

## Debugging

### Logs
Structured JSON logs via slog. Configure in `logging` section.

### Database
- Connect: `psql $ROUTER_DB_URL`
- View migrations: `goose -dir migrations status`

### Redis
- Connect: `redis-cli -u $ROUTER_REDIS_URL`

### React Query DevTools
Available in development mode in the browser.

## Resources

- [Architecture Decisions](../architecture/)
- [API Reference](../reference/)
- [Runtime Configuration](../runtime/)
