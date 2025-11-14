# Developer Guide

Everything required to work on Open Model Gateway locally.

## Prerequisites

- Go 1.25+
- Bun 1.1+ (manages the Vite/React frontend)
- Docker (optional but recommended for Postgres + Redis)
- `make`, `git`, `sqlc` and `goose` (install via `go install` if they are not already on your PATH)

## First-Time Setup

```bash
git clone https://github.com/.../open_model_gateway.git
cd open_model_gateway
cp docs/runtime/router.example.yaml deploy/router.local.yaml
make compose-up        # spins up Postgres (5432) + Redis (6379)
bun install --cwd backend/frontend
```

Update `deploy/router.local.yaml` with the secrets you want to use locally (there is no default admin password). The backend reads `ROUTER_CONFIG_FILE`, `ROUTER_DB_URL`, and `ROUTER_REDIS_URL`; the Makefile wires these to `deploy/router.local.yaml`, `postgres://...`, and `redis://...`.

## Running the Stack

The easiest workflow is:

```bash
make run-backend
```

This target:

1. Builds the frontend (`bun run build`) and copies the artifacts into `backend/internal/httpserver/ui/dist/`.
2. Runs `go run ./cmd/routerd` with the config/database/redis URLs from the Makefile variables.

The admin portal is available at `http://localhost:8090/admin`, the user portal at `/`, and OpenAI-compatible APIs under `/v1/*`.

To stop support services:

```bash
make compose-down
```

## Hot Reload Options

- Frontend: run `cd backend/frontend && bun run dev` for Vite HMR, and point API calls at `http://localhost:8090`.
- Backend: install [`air`](https://github.com/cosmtrek/air) or similar. A simple approach is to run `go run ./cmd/routerd` directly (without `make run-backend`) when you do not need to rebuild the UI.

## Smoke Tests

Use the seeded API key from `deploy/router.local.yaml` (or `ROUTER_BOOTSTRAP_*`) for quick verification.

### Images API (generate / edit / variation)

```bash
API_KEY="sk-demo.my-secret"

# Generation
curl http://localhost:8090/v1/images/generations \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
        "model": "gpt-image-1",
        "prompt": "A neon Go gopher piloting a bioluminescent submersible"
      }' | jq -r '.data[0].b64_json' | base64 --decode > neon-gopher.png

# Edit (mask optional)
curl http://localhost:8090/v1/images/edits \
  -H "Authorization: Bearer $API_KEY" \
  -F "model=gpt-image-1" \
  -F "prompt=Fill the blank canvas with a watercolor skyline" \
  -F "image=@./examples/base.png" \
  -F "mask=@./examples/mask.png" \
  -o edit.ndjson

# Variation
curl http://localhost:8090/v1/images/variations \
  -H "Authorization: Bearer $API_KEY" \
  -F "model=gpt-image-1" \
  -F "image=@./examples/base.png" \
  -F "n=2" \
  -o variation.ndjson
```

OpenAI + OpenAI-compatible adapters honor edits/variations; Azure, Bedrock, and Vertex handlers will return `image_operation_unsupported` so clients can fall back gracefully.

### Batches API

```bash
cat <<'EOF' > batch.sample.jsonl
{"custom_id":"smoke-1","method":"POST","url":"/v1/chat/completions","body":{"model":"gpt-5-mini","messages":[{"role":"system","content":"You are a haiku bot."},{"role":"user","content":"Write one about routers."}],"max_tokens":64}}
{"custom_id":"smoke-2","method":"POST","url":"/v1/chat/completions","body":{"model":"gpt-5-mini","messages":[{"role":"system","content":"You are a teacher."},{"role":"user","content":"Explain JSONL briefly."}],"max_tokens":64}}
EOF

FILE_ID=$(curl -s http://localhost:8090/v1/files \
  -H "Authorization: Bearer $API_KEY" \
  -F "purpose=batch" \
  -F "file=@batch.sample.jsonl" | jq -r '.id')

BATCH_ID=$(curl -s http://localhost:8090/v1/batches \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
        "input_file_id": "'"$FILE_ID"'",
        "endpoint": "/v1/chat/completions",
        "completion_window": "24h",
        "metadata": {"label": "dev smoke"}
      }' | jq -r '.id')

curl -s http://localhost:8090/v1/batches/$BATCH_ID -H "Authorization: Bearer $API_KEY" | jq
curl -s http://localhost:8090/v1/batches/$BATCH_ID/output -H "Authorization: Bearer $API_KEY" -o batch_${BATCH_ID}.jsonl
curl -s http://localhost:8090/v1/batches/$BATCH_ID/errors -H "Authorization: Bearer $API_KEY" -o batch_errors_${BATCH_ID}.jsonl
```

The user portal mirrors these operations; the Batches tab now defaults to a user’s personal tenant and keeps downloads in-page.

### Audio Text-to-Speech

```bash
curl http://localhost:8090/v1/audio/speech \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -o speech.mp3 \
  -d '{
        "model": "gpt-4o-mini-tts",
        "input": "Open Model Gateway now speaks!",
        "voice": "alloy",
        "format": "mp3"
      }'
```

The endpoint currently returns binary audio (mp3 by default). Streaming responses (`"stream": true`) are on the roadmap.

## Tests & Tooling

```bash
make test-backend         # go test ./... under backend/
cd backend && goose status
cd backend && sqlc generate
```

- `backend/sql/queries/*.sql` define the SQLC contract. Change queries, then run `sqlc generate`.
- Goose migrations live in `backend/migrations/`. Use `goose create name sql` and commit the generated file.
- The batch worker, providers, and HTTP handlers all have unit tests under `backend/internal/...`.

## Coding Standards

- Stick to Go 1.25 formatting (`gofmt` or `goimports`) and TypeScript’s ESLint rules (run `bun run lint` when available).
- Keep configuration additions documented in `docs/runtime/router.example.yaml` and `docs/runtime/config.md`.
- When adding a new provider, follow the checklist in `docs/architecture/providers/adding.md`.
- Frontend code uses the shadcn UI kit, React Query, and the app router pattern (`src/apps/{admin,user}`). Shared components live under `src/components` or `src/ui`.

## Useful Scripts

| Command | Description |
| --- | --- |
| `make compose-up` | Starts Postgres + Redis via Docker Compose. |
| `make build-ui` | Builds the frontend and copies assets into the embedded Go FS. |
| `make run-backend CONFIG=/path/to/config.yaml` | Runs routerd with a different config file. |
| `bun run generate-icons` | Example of the Bun ecosystem; adjust as needed. |

## Troubleshooting

- **Startup fails with missing migrations**: run `make compose-down && make compose-up` to recreate the database volumes, or point `database.migrations_dir` at the correct directory.
- **`authorization required` on `/user/*` endpoints**: ensure cookies are being sent. The UI relies on `httpOnly` admin/user session cookies; direct `curl` calls require a `Bearer` token.
- **Frontend changes not visible**: rerun `make run-backend` or `make build-ui` so the embedded assets are refreshed.
- **`address already in use :8090`**: stop stray router processes via `pkill -f cmd/routerd` or identify the PID with `lsof -i tcp:8090` / `kill <pid>` before rerunning `make run-backend`.

Feel free to add more scripts/targets in the Makefile as the project grows.
