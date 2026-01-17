# System administrator install guide

Use this guide to install, configure, and upgrade Open Model Gateway in production. It consolidates the deploy paths (release bundle, Docker, Kubernetes) plus the first-boot checklist for bringing the admin and user portals online.

## Who this is for

- System administrators responsible for infrastructure, secrets, and runtime configuration.
- Platform operators who manage Postgres/Redis, observability sinks, and provider credentials.

## Portal and API endpoints

| Surface | Default path | Notes |
| --- | --- | --- |
| Admin portal | `/admin/ui` | Requires admin login; includes catalog, tenants, budgets, provider health. |
| User portal | `/` | Standard user + tenant admin portal (personal tenants live here). |
| Public API | `/v1/*` | OpenAI-compatible surface for workloads. |
| Admin API | `/admin/*` | Automation endpoints for tenants, keys, budgets, providers. |
| User API | `/user/*` | Tenant-scoped usage, keys, and membership endpoints. |
| Health | `/healthz` | JSON health checks for DB, Redis, and providers. |
| Metrics | `/metrics` | Prometheus metrics (enable via `observability.enable_metrics`). |

![TODO: Admin portal login screen](../assets/screenshots/admin-login.png)

## Prerequisites

| Component | Requirement | Notes |
| --- | --- | --- |
| Go toolchain (optional) | 1.25+ | Only required when building from source. |
| Postgres | 14+ | Stores tenants, usage, budgets, incidents. |
| Redis | 7+ | Rate limits, idempotency, provider health cache. |
| Object storage | Local disk or S3 | Required if `/v1/files` or batch outputs are enabled. |
| OTEL collector | Optional | Enables tracing + metrics exports. |

## Prepare configuration

1. Start from `deploy/router.example.yaml` and trim unused providers and model aliases.
2. Set `public.base_url` to the external hostname so invites and alerts use correct links.
3. Update `admin.oidc.redirect_url` to the admin callback endpoint (`https://<host>/admin/ui/auth/oidc/callback`) when enabling SSO.
4. Store secrets in your secret manager and inject via ENV rather than committing them to YAML.
5. Enable `database.run_migrations` for single-instance installs, or run migrations separately before scaling out.

Common environment variables:

```bash
export ROUTER_CONFIG_FILE=/etc/open-model-gateway/router.yaml
export ROUTER_DB_URL=postgres://user:pass@db:5432/open_gateway?sslmode=disable
export ROUTER_REDIS_URL=redis://redis:6379/0
export ROUTER_ADMIN_SESSION_JWT_SECRET=replace-me
```

Reference config keys in `docs/admin/runtime/config.md` and the annotated sample in `docs/admin/runtime/router-example.md`.

## Install path A: release bundle + systemd

1. Download the latest `open-model-gateway_<tag>_<os>_<arch>.tar.gz` from GitHub Releases.
2. Extract into `/opt/open-model-gateway` (or your preferred location).
3. Copy `deploy/router.example.yaml` to `/etc/open-model-gateway/router.yaml` and edit it.
4. Add required environment variables (DB, Redis, provider keys, JWT secret).
5. Start the router with systemd or your process supervisor.

Example systemd unit:

```ini
[Unit]
Description=Open Model Gateway
After=network.target

[Service]
WorkingDirectory=/opt/open-model-gateway
ExecStart=/opt/open-model-gateway/router
Environment=ROUTER_CONFIG_FILE=/etc/open-model-gateway/router.yaml
Environment=ROUTER_DB_URL=postgres://user:pass@db:5432/open_gateway?sslmode=disable
Environment=ROUTER_REDIS_URL=redis://redis:6379/0
Environment=ROUTER_ADMIN_SESSION_JWT_SECRET=replace-me
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

## Install path B: Docker Compose

1. Update `deploy/router.example.yaml` and save a copy as `deploy/router.local.yaml`.
2. For a production image, uncomment the `router` service in `deploy/docker-compose.yml`.
3. For local builds, use the dev stack:

```bash
cd deploy
docker compose -f docker-compose.dev.yml up --build -d
```

4. Verify the router listens on `:8090` and the admin portal loads at `/admin/ui`.

![TODO: Admin dashboard health cards](../assets/screenshots/admin-dashboard.png)

## Install path C: Kubernetes

Use ConfigMaps for non-secret config, Secrets for credentials, and a persistent volume for files when `files.storage=local`.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: omg-config
data:
  router.yaml: |
    server:
      listen_addr: ":8090"
    public:
      base_url: "https://gateway.example.com"
    database:
      run_migrations: false
      url: "postgres://user:pass@db:5432/open_gateway?sslmode=disable"
    redis:
      url: "redis://redis:6379/0"
    files:
      storage: "s3"
    admin:
      session:
        jwt_secret: "replace-me"
---
apiVersion: v1
kind: Secret
metadata:
  name: omg-secrets
stringData:
  ROUTER_DB_URL: "postgres://user:pass@db:5432/open_gateway?sslmode=disable"
  ROUTER_REDIS_URL: "redis://redis:6379/0"
  ROUTER_ADMIN_SESSION_JWT_SECRET: "replace-me"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: omg-router
spec:
  replicas: 2
  selector:
    matchLabels:
      app: omg-router
  template:
    metadata:
      labels:
        app: omg-router
    spec:
      containers:
        - name: router
          image: ghcr.io/ncecere/open_model_gateway:v0.4.0
          ports:
            - containerPort: 8090
          env:
            - name: ROUTER_CONFIG_FILE
              value: /config/router.yaml
            - name: ROUTER_DB_URL
              valueFrom:
                secretKeyRef:
                  name: omg-secrets
                  key: ROUTER_DB_URL
            - name: ROUTER_REDIS_URL
              valueFrom:
                secretKeyRef:
                  name: omg-secrets
                  key: ROUTER_REDIS_URL
            - name: ROUTER_ADMIN_SESSION_JWT_SECRET
              valueFrom:
                secretKeyRef:
                  name: omg-secrets
                  key: ROUTER_ADMIN_SESSION_JWT_SECRET
          volumeMounts:
            - name: config
              mountPath: /config/router.yaml
              subPath: router.yaml
          readinessProbe:
            httpGet:
              path: /healthz
              port: 8090
      volumes:
        - name: config
          configMap:
            name: omg-config
---
apiVersion: v1
kind: Service
metadata:
  name: omg-router
spec:
  selector:
    app: omg-router
  ports:
    - port: 80
      targetPort: 8090
```

Kubernetes notes:

- Run Goose migrations as a one-off Job before scaling to multiple replicas.
- Use an Ingress to route `/admin/ui`, `/`, `/v1/*`, `/admin/*`, and `/user/*` to the service.
- If `files.storage=local`, mount a PVC at the configured directory.

## First boot checklist

1. Confirm `GET /healthz` returns `status: ok`.
2. Log in to the admin portal with bootstrap credentials from `bootstrap.admin_users`.
3. Verify provider status under **Admin -> Providers**.
4. Open the user portal at `/` and confirm your personal tenant appears under **Tenants**.
5. Call `GET /v1/models` using a bootstrap API key.

```bash
curl -sS http://localhost:8090/v1/models \
  -H "Authorization: Bearer sk-demo.my-secret" | jq
```

![TODO: Providers health page](../assets/screenshots/admin-providers-health.png)

## Upgrade checklist

1. Back up Postgres and any local file storage.
2. Deploy the new binary or container image.
3. Run migrations once (either `database.run_migrations: true` or a manual Goose run).
4. Rebuild embedded UI with `make build-ui` when building from source.
5. Verify `/healthz`, `/metrics`, and portal login flows.

## Next steps

- Day-2 operations: `docs/admin/guide.md`
- Runtime configuration reference: `docs/admin/runtime/README.md`
- Troubleshooting: `docs/reference/troubleshooting.md`
