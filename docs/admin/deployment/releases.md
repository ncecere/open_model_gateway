# Release & Packaging Guide

Push a tag matching `v*` to publish two artifacts.

| Artifact | Summary |
| --- | --- |
| Router bundle (tar.gz) | Compiled `router` binary, database migrations, and sample config for direct host installs. |
| Docker image | Multi-stage build pushed to `ghcr.io/ncecere/open_model_gateway:<tag>` and `:latest` for containerized deployments. |

## Plan release workflow

Push a tag like `v0.4.0` to trigger `.github/workflows/release.yml`.
GitHub Actions then runs the jobs below.

| Job | Purpose | Key actions |
| --- | --- | --- |
| build | Compile and verify the release bundle. | Check out the repo, install Bun and Go, run `make build-ui`, run `make test-backend`, build `cmd/routerd`, and package the tarball artifact. |
| docker | Publish the container image. | Build the multi-stage Dockerfile and push tags to GHCR (`<tag>` and `latest`). |
| release | Publish the GitHub release. | Download the tarball artifact and attach it to the tagged release page. |

If any job fails the release is blocked.

## Use binary build

Run the router directly on a host with the tarball bundle.

1. Download `open-model-gateway_<tag>_linux_amd64.tar.gz` from the Releases page.
2. Extract the bundle.

   ```bash
   tar -xzf open-model-gateway_<tag>_linux_amd64.tar.gz -C /opt/open-model-gateway
   ```

3. Edit the extracted `deploy/router.local.yaml` to match your Postgres, Redis, and provider credentials.
4. Start the router.

   ```bash
   cd /opt/open-model-gateway
   ROUTER_CONFIG_FILE=/path/to/router.yaml ./router
   ```

The router runs migrations on boot when `database.run_migrations: true`.
Disable it if you manage migrations with Goose using `backend/migrations`.

## Use Docker image

Run the published GHCR image when you prefer containers.

Pull the tagged image.

```bash
docker pull ghcr.io/ncecere/open_model_gateway:v0.4.0
```

Run the container with explicit config mounts and environment variables.

```bash
docker run --rm \
  -p 8090:8090 \
  -v ./router.local.yaml:/config/router.yaml:ro \
  -e ROUTER_CONFIG_FILE=/config/router.yaml \
  -e ROUTER_DB_URL=postgres://user:pass@postgres:5432/open_gateway?sslmode=disable \
  -e ROUTER_REDIS_URL=redis://redis:6379/0 \
  ghcr.io/ncecere/open_model_gateway:v0.4.0
```

Use `deploy/docker-compose.yml` for the router plus Postgres, Redis, and OTEL.

```bash
cd deploy
docker compose build router
docker compose up -d
```

Stop the stack with `docker compose down`.

## Cut a release

Follow this checklist when shipping a new version.

1. Confirm `main` is green and contains the commits you need.
2. Tag and push the release: `git tag v0.4.0 && git push origin v0.4.0`.
3. Monitor the Release workflow until it finishes.
4. Confirm GitHub Releases shows the tarball and GHCR exposes `<tag>` plus `latest`.

Deploy from whichever artifact fits your environment.
