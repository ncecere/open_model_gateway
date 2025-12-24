# Observability Guide

Instrument the gateway with OTEL traces and Prometheus metrics so routing, budgeting, and provider telemetry stay visible.

## Track telemetry metrics
Scrape `/metrics` (enabled via `observability.enable_metrics`) for these series.

| Metric | Type | Labels | Notes |
| --- | --- | --- | --- |
| `open_model_gateway_http_requests_total` | Counter | `route`, `status`, `tenant_scope` | Total HTTP calls across `/v1` and `/admin`. |
| `open_model_gateway_http_request_duration_seconds` | Histogram | `route`, `status` | Latency buckets aligned with API SLIs. |
| `open_model_gateway_api_provider_retries_total` | Counter | `provider`, `model_alias`, `reason` | Executor retry counts per adapter. |
| `open_model_gateway_rate_limiter_parallel_inflight` | Gauge | `tenant_id`, `api_key_id` | In-flight concurrent request counts. |
| `open_model_gateway_budget_evaluation_duration_seconds` | Histogram | `tenant_id` | Tracks time spent computing budgets. |
| `gateway_provider_request_duration_seconds` | Histogram | `provider`, `model_alias`, `route`, `result` | Upstream latency per provider (sync + stream completion). |
| `gateway_provider_requests_total` | Counter | `provider`, `model_alias`, `route`, `result` | Provider request outcomes. |
| `gateway_provider_tokens_total` | Counter | `provider`, `model_alias`, `route`, `direction` | Token usage emitted after each call/stream. |
| `gateway_provider_upstream_errors_total` | Counter | `provider`, `model_alias`, `route`, `class` | Transport/auth/timeout buckets for error triage. |
| `gateway_provider_active_streams` | Gauge | `provider`, `model_alias`, `route` | Optional concurrency watcher for streaming load. |

## Stamp OTEL spans
Enable `observability.enable_otlp` and point `observability.otlp_endpoint` at a collector; spans are emitted for `open-model-gateway/http`, `open-model-gateway/executor`, `open-model-gateway/ratelimiter`, and `open-model-gateway/budget-evaluator` with attributes such as `provider`, `model_alias`, `deployment`, `region`, `input_tokens`, and `output_tokens`.

## Evaluate SLIs
Provider telemetry evaluates latency/error windows on the cadence defined below.

| Config key | Default | Description |
| --- | --- | --- |
| `telemetry.provider.evaluation_interval` | 60s | How often the evaluator checks metrics. |
| `telemetry.provider.window_size` | 5m | Sliding window for SLI calculations. |
| `telemetry.provider.latency_p95_ms` | 5000 | Max allowed p95 latency before incidents open. |
| `telemetry.provider.error_rate` | 0.1 | Max allowed error rate (10%). |
| `telemetry.provider.timeout_rate` | 0.05 | Max allowed timeout rate (5%). |
| `telemetry.provider.min_samples` | 50 | Minimum requests required before evaluating. |
| `telemetry.provider.downweight_when_degraded` | true | Whether routing weights drop for degraded entries. |
| `telemetry.provider.incident_retention_days` | 30 | Retention window for incident history. |

## Feed routing decisions
The router stores health state in Redis, exposes incidents via `/admin/providers`, and down-weights degraded catalog entries whenever telemetry or active probes fail, while single-entry aliases remain advisory-only.

## Run local collectors
Use `deploy/docker-compose.yml` or `deploy/otel-collector.yaml` to run Postgres, Redis, and an OTLP collector locally; `make run-backend` enables `/metrics` and traces once `observability.enable_metrics` or `observability.enable_otlp` are set.

## Extend coverage
Upcoming work (tracked in `agents.md`) includes surfacing batch queue depth, sweeper stats, richer budget history, and publishing ready-to-use Grafana dashboards plus Kubernetes collector manifests—add metrics there as new services land.
