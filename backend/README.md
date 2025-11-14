# Backend Developer Guide

## Tooling
- Go 1.25+
- Goose (installed via `go install github.com/pressly/goose/v3/cmd/goose@latest`)
- sqlc (installed via `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`)
- Postgres + Redis (see root project compose file when available)

Ensure `$GOPATH/bin` is on your `PATH` so `goose` and `sqlc` binaries are discoverable.

## Migrations

Migrations live in `migrations/` and use Goose timestamped files.

- Create a new migration:
  ```bash
  goose -dir migrations create add_new_table sql
  ```
- Apply migrations against a database:
  ```bash
  goose -dir migrations postgres "$ROUTER_DB_URL" up
  ```
- Check status:
  ```bash
  goose -dir migrations postgres "$ROUTER_DB_URL" status
  ```

The initial schema is mirrored under `sql/schema/` for sqlc code generation.

## SQLC Code Generation

Queries live in `sql/queries/*.sql`. After editing or adding queries, regenerate Go types:

```bash
sqlc generate
```

Generated code is written to `internal/db`. Run `go mod tidy` if new dependencies (e.g., `decimal`) are introduced.

## Testing & Coverage

The backend test suite runs entirely with `go test` and uses lightweight fakes/mocks so no third-party provider calls are made:

```bash
cd backend && go test ./...
```

- Router engine tests assert circuit breakers and weighted selection (`internal/router/engine_test.go`).
- Rate limiter tests spin up an in-memory Redis (miniredis) to verify RPM/parallel/token semantics (`internal/limits/limiter_test.go`).
- Batch worker helpers (error encoding, TTL math, status mapping) are covered under `internal/batchworker/`.
- Usage service tests validate timezone-aware buckets and multi-entity deduplication.
- Provider contract tests replay captured fixtures for Azure, Bedrock, and Vertex adapters to ensure OpenAI-compatible responses (`internal/adapters/**/adapter_contract_test.go`, fixtures live under `internal/providers/fixtures`).

CI and local contributors should run `make test-backend` (defined at the repo root) before opening PRs to ensure regressions are caught.

## Configuration Loader

The configuration package loads YAML files (default `router.yaml` / `config/router.yaml`) with `.env` overlay. Required environment variables include `ROUTER_DB_URL` and `ROUTER_REDIS_URL`; defaults for rate limits and budgets can be overridden via ENV (`DEFAULT_TPM`, etc.).

Database-specific knobs:

- `DATABASE_RUN_MIGRATIONS` (bool, default `true`) to toggle Goose execution on boot
- `DATABASE_MIGRATIONS_DIR` (default `./migrations`) to point at migration files when running from a custom working directory
- `DATABASE_MAX_CONNS`, `DATABASE_MIN_CONNS`, `DATABASE_MAX_CONN_IDLE_TIME`, `DATABASE_MAX_CONN_LIFETIME`

Redis tuning:`REDIS_DB`, `REDIS_POOL_SIZE` (pool sizing overrides when not relying on defaults).

### Admin Auth Settings

The admin surface supports both local credentials and OIDC sign-in.

- `ADMIN_SESSION_JWT_SECRET` (required) – HMAC secret for access/refresh tokens.
- `ADMIN_SESSION_ACCESS_TOKEN_TTL` (default `15m`)
- `ADMIN_SESSION_REFRESH_TOKEN_TTL` (default `24h`)
- `ADMIN_SESSION_COOKIE_NAME` (default `og_admin_session`)
- `ADMIN_LOCAL_ENABLED` (`true` by default) – disable to enforce SSO-only.
- `ADMIN_OIDC_ENABLED`, `ADMIN_OIDC_ISSUER`, `ADMIN_OIDC_CLIENT_ID`, `ADMIN_OIDC_CLIENT_SECRET`, `ADMIN_OIDC_REDIRECT_URL`, `ADMIN_OIDC_SCOPES`, `ADMIN_OIDC_ALLOWED_DOMAINS`.

On startup, the backend wires password hashing (Argon2id), JWT issuance, and OIDC discovery. Use the SQLC-generated helpers (`UpsertCredential`, `GetCredentialByUserAndProvider`) to seed system admin accounts or rotate secrets.

## Running Locally (WIP)

A full local stack (Postgres, Redis, backend, frontend) will be documented once Docker Compose files are added. For now, you can run the server with:

```bash
ROUTER_DB_URL=postgres://... ROUTER_REDIS_URL=redis://... go run ./cmd/routerd
```

The service exposes `GET /healthz` by default.

### Admin Auth HTTP Surface

The backend currently serves:

- `GET /admin/auth/methods` – feature discovery (`["local","oidc"]` etc.).
- `POST /admin/auth/login` – local credential login (JSON body `{"email","password"}`).
- `GET /admin/auth/oidc/start` and `GET /admin/auth/oidc/callback` – OIDC authorization code flow helper endpoints (state persisted in Redis).
- `POST /admin/auth/refresh` – swaps a refresh token (cookie or body) for a new access token.
- `POST /admin/auth/logout` – clears the refresh-token cookie.

Refresh tokens are issued as HTTP-only cookies (`ADMIN_SESSION_COOKIE_NAME`), while access tokens are returned in the JSON payload.
