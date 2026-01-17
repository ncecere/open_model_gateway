# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Open Model Gateway is a Go/Fiber inference router that speaks the OpenAI API while adding tenant isolation, multi-provider routing, budget controls, and usage metering. It proxies requests to OpenAI, Azure OpenAI, Anthropic, AWS Bedrock, Vertex AI, OpenRouter, Groq, and vLLM.

## Common Commands

### Development
```bash
make compose-up              # Start Postgres + Redis containers
make compose-down            # Stop containers and remove volumes
make run-backend             # Build UI and run router (uses deploy/router.local.yaml)
make run-backend CONFIG=/path/to/custom.yaml  # Run with custom config
```

### Testing
```bash
make test-backend            # Run all Go tests (cd backend && go test ./...)
cd backend && go test ./internal/adapters/...  # Test specific package
cd backend && go test -run TestFoo ./internal/app  # Run single test
```

### Frontend
```bash
cd backend/frontend && bun install   # Install frontend deps
cd backend/frontend && bun run dev   # Dev server with HMR
cd backend/frontend && bun run build # Production build
cd backend/frontend && bun run test  # Run Vitest tests
```

### Database & Code Generation
```bash
# Create new migration (run from backend/)
goose -dir migrations create feature_name sql

# Regenerate SQLC types after changing sql/queries/*.sql
cd backend && sqlc generate
```

## Architecture

### Backend Structure (`backend/`)

- **cmd/routerd/main.go**: Entry point - initializes runtime, starts HTTP server and background jobs
- **internal/runtime/**: Application bootstrap - loads config, runs migrations, builds the Container
- **internal/app/container.go**: Dependency injection container holding all services, DB pool, Redis, providers
- **internal/config/**: YAML config parsing with Viper, merged with `ROUTER_*` env vars
- **internal/httpserver/**: Fiber HTTP routes split into:
  - `public/`: OpenAI-compatible `/v1/*` endpoints (chat, embeddings, images, audio, files, batches)
  - `admin/`: `/admin/*` management APIs (tenants, users, models, budgets, rate limits)
  - `user/`: `/user/*` self-service APIs for tenant members
- **internal/adapters/**: Provider adapters (openai, anthropic, azureopenai, bedrock, vertex, groq, openrouter, vllm) implementing common interfaces for chat, embeddings, etc.
- **internal/providers/**: Factory for constructing adapters from config
- **internal/router/**: Request routing engine with weighted selection, failover, and health-aware routing
- **internal/services/**: Business logic services (tenant, usage, batches, files, admin operations)
- **internal/db/**: SQLC-generated query layer (do not edit directly - regenerate with `sqlc generate`)
- **internal/limits/**: Rate limiting with Redis-backed counters
- **internal/observability/**: OTEL tracing/metrics setup, Prometheus exporter

### Frontend Structure (`backend/frontend/`)

React/Vite with TypeScript, Tailwind CSS, and shadcn/ui components. Two portals:
- Admin portal (`/admin/`): Tenant management, model catalog, provider health, usage dashboards
- User portal (`/`): Self-service key management, usage views, budget status

### Data Layer

- **Postgres**: Tenants, users, API keys, usage records, budgets, rate limits, audit logs
- **Redis**: Rate limit counters, idempotency cache, provider health snapshots
- **Migrations**: Goose migrations in `backend/migrations/`, auto-run on boot if `database.run_migrations: true`
- **SQLC**: Queries in `backend/sql/queries/*.sql`, schema in `backend/sql/schema/`

## Configuration

Primary config: `deploy/router.local.yaml` (copy from `deploy/router.example.yaml`)

Key sections:
- `server`: Listener address, timeouts, body limits
- `database`: Postgres connection, pool size, migration toggle
- `redis`: Connection for rate limits and caching
- `providers.<slug>`: API keys and settings for each upstream provider
- `model_catalog`: Model aliases, provider bindings, pricing, routing weights
- `rate_limits` / `budgets`: Default limits with per-tenant/per-key overrides
- `bootstrap`: Declarative seeding of tenants, users, API keys (idempotent on restart)

Environment overrides use `ROUTER_` prefix (e.g., `ROUTER_DB_URL`, `ROUTER_PROVIDERS_OPENAI_KEY`).

## Key Patterns

- Provider adapters implement interfaces in `internal/providers/` - add new providers by creating adapter in `internal/adapters/<name>/`
- All SQL queries go through SQLC - modify `sql/queries/*.sql` then run `sqlc generate`
- Rate limits and budgets are enforced per-request via middleware in the public routes
- Streaming responses use SSE with proper idle timeout handling
- Usage is logged asynchronously to Postgres with cost computation based on token counts and model pricing
