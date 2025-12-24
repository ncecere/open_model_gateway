---
title: observability setup
description: Enable metrics and OTEL exports
---
[**Open Model Gateway**](/) exposes Prometheus metrics and OTLP traces that feed local or cluster collectors.

---

#### Enable exporters
Set the following in `router.yaml` (or matching `ROUTER_OBSERVABILITY_*` env vars) to turn on metrics and tracing.

```yaml
observability:
  enable_metrics: true
  enable_otlp: true
  otlp_endpoint: "otel-collector:4317"
```

Metrics emit on `/metrics` and the OTLP exporter batches spans to the configured gRPC endpoint.

---

#### Run local collector
`deploy/docker-compose.yml` ships an `otel-collector` service; start it with Postgres and Redis:

```bash
docker compose -f deploy/docker-compose.yml up -d postgres redis otel-collector
```

The collector listens on `4317/4318` and uses `deploy/otel-collector.yaml` to print spans to logs for quick inspection.

---

#### Deploy on clusters
Apply `deploy/otel-collector.yaml` to create a dedicated collector Deployment + Service, then add these env vars to the gateway pods:

```yaml
env:
  - name: ROUTER_OBSERVABILITY_ENABLE_OTLP
    value: "true"
  - name: ROUTER_OBSERVABILITY_OTLP_ENDPOINT
    value: "otel-collector.observability:4317"
```

Ensure the gateway namespace can reach the collector Service through network policies or mesh rules.

---

#### Verify signals
1. Hit `/metrics` and confirm counters such as `open_model_gateway_http_requests_total` increment under load.
2. Send traffic (`curl /v1/chat/completions`) and check collector logs or your OTEL backend for new spans.
3. Review telemetry dashboards (Prometheus, Jaeger, Tempo, etc.) to confirm health data mirrors provider status.

---

#### Research
- Adapted instructions from `docs/runtime/observability.md` and deployment manifests in `deploy/otel-collector*.yaml`.
- Confirmed OTLP env overrides and metric names via recent observability work logged in `AGENTS.md`.
