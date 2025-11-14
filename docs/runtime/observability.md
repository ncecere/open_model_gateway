# Observability Setup

The gateway can export Prometheus metrics and OTLP traces. This guide shows how to enable those features locally and in Kubernetes.

## 1. Enable Metrics / Traces in `router.yaml`

```yaml
observability:
  enable_metrics: true             # exposes /metrics
  enable_otlp: true                # enable OTLP exporter
  otlp_endpoint: "otel-collector:4317"  # gRPC endpoint of your collector
```

Metrics are served from `/metrics` once `enable_metrics` is true. The OTLP exporter batches spans and delivers them to the endpoint above.

## 2. Local Collector via Docker Compose

`deploy/docker-compose.yml` now ships an `otel-collector` service. Bring it up alongside Postgres/Redis:

```bash
docker compose -f deploy/docker-compose.yml up -d postgres redis otel-collector
```

The collector listens on `4317` (gRPC) and `4318` (HTTP). Its config lives in `deploy/otel-collector.yaml` and ships received traces to the logging exporter so you can inspect spans from the terminal.

Run the gateway with `observability.enable_otlp=true` and it will emit spans to the collector automatically.

## 3. Kubernetes Manifest

For clusters, the `deploy/otel-collector.yaml` manifest creates a standalone collector Deployment + Service. Apply it first:

```bash
kubectl apply -f deploy/otel-collector.yaml
```

Then configure your gateway Deployment/StatefulSet with:

```yaml
env:
  - name: ROUTER_OBSERVABILITY_ENABLE_OTLP
    value: "true"
  - name: ROUTER_OBSERVABILITY_OTLP_ENDPOINT
    value: "otel-collector.observability:4317"
```

Make sure the gateway pod can resolve/reach the collector Service.

## 4. Verifying

1. Hit `/metrics` – you should see `open_model_gateway_http_requests_total` and `open_model_gateway_http_request_duration_seconds`.
2. Generate traffic (e.g. `curl /v1/chat/completions`).
3. Check collector logs (`docker logs open-model-gateway-otel-collector` or `kubectl logs`) for span exports.

From here you can wire the collector exporters to your preferred backend (Jaeger, Tempo, OTLP/HTTP, etc.).
