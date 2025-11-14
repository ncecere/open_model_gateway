# Release & Packaging Guide

This project ships two official artifacts every time a tag matching `v*` is pushed:

1. **Router bundle (tar.gz)** – contains the compiled `router` binary, database migrations, and the sample config.  Download it from the GitHub Releases page and run it directly on a host.
2. **Docker image** – built from the root `Dockerfile` and published to the GitHub Container Registry (GHCR) as `ghcr.io/ncecere/open_model_gateway:<tag>` and `:latest`.

## How the Release Workflow Works

The GitHub Actions workflow defined in [`.github/workflows/release.yml`](../../.github/workflows/release.yml) is triggered every time a tag prefixed with `v` is pushed (for example `v0.4.0`). It performs three jobs:

1. **build**
   - Checks out the repository.
   - Installs Bun and Go.
   - Runs `make build-ui` to compile the React/Vite admin and user portals.
   - Executes `make test-backend` to run Go unit tests.
   - Builds the `router` binary (`go build ./cmd/routerd`).
   - Packages the binary, migrations, and `deploy/router.local.yaml` into `open-model-gateway_<tag>_linux_amd64.tar.gz` and uploads it as an artifact.
2. **docker**
   - Builds the multi-stage image defined in the root `Dockerfile`.
   - Pushes the resulting image to GHCR under both the version tag and the rolling `latest` tag.
3. **release**
   - Downloads the packaged tarball.
   - Publishes a GitHub Release for the tag and attaches the tarball.

If any of those jobs fail the release is blocked.

## Consuming the Binary

1. Download `open-model-gateway_<tag>_linux_amd64.tar.gz` from the Releases page.
2. Unpack it somewhere on your server:

   ```bash
   tar -xzf open-model-gateway_<tag>_linux_amd64.tar.gz -C /opt/open-model-gateway
   ```

3. Edit the extracted `deploy/router.local.yaml` (or supply your own config) to point at your Postgres/Redis instances and provider credentials.
4. Start the router:

   ```bash
   cd /opt/open-model-gateway
   ROUTER_CONFIG_FILE=/path/to/router.yaml ./router
   ```

By default the router runs migrations on boot (`database.run_migrations: true`). If you manage migrations externally, disable that flag and run Goose manually using the bundled `backend/migrations` directory.

## Consuming the Docker Image

Each release publishes `ghcr.io/ncecere/open_model_gateway:<tag>` (e.g., `ghcr.io/ncecere/open_model_gateway:v0.4.0`) and keeps `ghcr.io/ncecere/open_model_gateway:latest` in sync. Pull it with:

```bash
docker pull ghcr.io/ncecere/open_model_gateway:v0.4.0
```

To run the container standalone you need to provide Postgres/Redis URLs and a config file. Example:

```bash
docker run --rm \
  -p 8090:8090 \
  -v ./router.local.yaml:/config/router.yaml:ro \
  -e ROUTER_CONFIG_FILE=/config/router.yaml \
  -e ROUTER_DB_URL=postgres://user:pass@postgres:5432/open_gateway?sslmode=disable \
  -e ROUTER_REDIS_URL=redis://redis:6379/0 \
  ghcr.io/ncecere/open_model_gateway:v0.4.0
```

For a fuller local environment (router + Postgres + Redis + OTEL), use `deploy/docker-compose.yml`. Edit `deploy/router.local.yaml` with your bootstrap data, then run:

```bash
cd deploy
docker compose build router
docker compose up -d
```

The router service binds host port `8090`. Stopping the stack is as simple as `docker compose down`.

## Cutting a Release

1. Ensure `main` is green and contains the changes you want to release.
2. Tag the commit: `git tag v0.4.0 && git push origin v0.4.0`.
3. Monitor the "Release" GitHub Action. When it finishes you’ll have:
   - A published GitHub Release with the tarball.
   - Docker images available in GHCR under the new tag and `latest`.

That’s it—deploy from the binary or the container, depending on your environment.
