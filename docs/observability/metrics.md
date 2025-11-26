# Provider Telemetry Metrics

These names are reserved for the provider telemetry workstream so OTEL + Prometheus stay consistent. The admin UI will consume summaries derived from the same series; Grafana can scrape `/metrics` directly (no proxy required).

## Metric Names (Prometheus)

| Metric | Type | Labels | Notes |
| --- | --- | --- | --- |
| `gateway_provider_request_duration_seconds` | Histogram | `provider`, `model_alias`, `route`, `result` (`success`, `error`, `timeout`, `canceled`) | Measures upstream latency (sync + stream finalization). Buckets align with p95 SLI checks. |
| `gateway_provider_requests_total` | Counter | `provider`, `model_alias`, `route`, `result` | Request outcome counts. `result=error` is set for upstream 4xx/5xx/transport failures; `timeout` reserved for client/server timeouts. |
| `gateway_provider_tokens_total` | Counter | `provider`, `model_alias`, `route`, `direction` (`input`, `output`) | Token usage emitted once per completed call/stream. |
| `gateway_provider_upstream_errors_total` | Counter | `provider`, `model_alias`, `route`, `class` (`transport`, `4xx`, `5xx`, `timeout`) | Keeps error taxonomy visible without inspecting logs. |
| `gateway_provider_active_streams` | Gauge | `provider`, `model_alias`, `route` | Optional gauge for in-flight streams to watch saturation. |

## OTEL Span/Attribute Conventions

- Span name: `provider.call` with attributes `provider`, `model_alias`, `route`, `provider_model`, `deployment`, `region`, `stream` (bool), `request.id`.
- Error attributes: `upstream.status_code`, `upstream.error_code`, `upstream.error_class` (`transport`, `timeout`, `auth`, `rate_limit`, `server`).
- Usage attributes: `input_tokens`, `output_tokens`, `billing_currency`, `billing_cost_micros`.

## SLI Windows & Incidents

- Evaluator runs every `telemetry.provider.evaluation_interval` across a sliding `window_size`.
- Threshold defaults (env aliases `ROUTER_TELEMETRY_PROVIDER_*` or `DEFAULT_PROVIDER_TELEMETRY_*`): `latency_p95_ms=5000`, `error_rate=0.1`, `timeout_rate=0.05`, `min_samples=50`.
- Incidents persist for `incident_retention_days` (default 30, configurable via config/admin settings). Open incidents are deduped per `provider/model_alias/incident_type`.

## Routing Behavior

When multiple catalog entries back the same alias, the router down-weights entries flagged unhealthy by the telemetry evaluator until they recover. Single-entry aliases remain advisory-only. Default is on; toggle via `telemetry.provider.downweight_when_degraded`.
