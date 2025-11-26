# Provider Telemetry & Alerting Tasks

Backlog ordered roughly by dependency. Owners are placeholders; adjust per sprint planning.

## Foundations
- [x] Backend: define telemetry config structs/env keys (`DEFAULT_PROVIDER_TELEMETRY_*`) and wire into config loader; add defaults doc.
- [x] Backend: add Goose migration for `provider_incidents` table (provider/model, type, opened_at, resolved_at, window, sample_error, counts).
- [x] Backend: add Redis key scheme for rolling windows (provider/model + metric + window) with TTL.
- [x] DevOps: pick OTEL/Prom metric names/conventions and publish in `docs/observability/metrics.md`.

## Instrumentation
- [x] Backend: wrap provider capability interfaces (chat/embed/image/audio/health) to emit histograms/counters for latency, upstream status, tokens, timeouts; ensure SSE helper flushes metrics on close/error. (health recorder wired)
- [x] Backend: add standardized error classification (timeout, 4xx, 5xx, transport) to label metrics and incidents.
- [x] Backend: expose new metrics to OTEL exporter and Prometheus `/metrics` with provider/model labels; add sampling guardrails if label count rises.
- [x] Testing: fake provider adapter that emits controlled latencies/errors for integration tests.

## SLI Evaluation & Alerts
- [x] Backend: implement SLI profiles (defaults + per-provider overrides) with thresholds for p95 latency, error rate %, timeout rate. (defaults wired; overrides config added but not yet populated)
- [x] Backend: collector to append per-request samples into Redis rolling windows (buckets per minute); background evaluator processes windows on interval.
- [x] Backend: incident manager opens/resolves incidents in Postgres, deduping per provider/model/type and attaching sample errors.
- [x] Backend: alert dispatcher reuses email/webhook channels; add per-channel cooldowns and include incident context (metrics, links).
- [x] Testing: unit tests for SLI math, window rollups, and alert dedupe paths.

## API & Router Integration
- [x] Backend: expose admin API endpoints for current SLIs, incidents, and active alerts; paginate/filter by provider/model/time. (incidents endpoint stubbed)
- [x] Backend: feed telemetry health into routing/health monitor (advisory flag) with audit logging when used in decisions.
- [x] Backend: seed dev data path to populate sample incidents for frontend/local demo.

## Frontend (Admin UI)
- [x] Frontend: add telemetry client hooks (fetch SLIs/incidents) under `src/apps/admin`.
- [x] Frontend: build Provider Health page with latency/error charts, incident list, and SLI threshold badges; reuse existing UI kit + React Query.
- [x] Frontend: add incident detail drawer showing sample errors and alert deliveries.
- [x] Frontend: add toggles/filters for provider/model/time window; ensure mobile responsiveness.
- [x] Testing: add component tests for charts/incident list states (loading, empty, error).

## Docs & Ops
- [x] Docs: update `docs/runtime/config.md` and provider docs with telemetry/alerting knobs and examples.
- [ ] Docs: add Grafana dashboards JSON and notes on wiring OTEL → Prometheus → Grafana.
- [ ] DevOps: extend `deploy/docker-compose.yml` to expose Prometheus scrape targets if not already, and ensure evaluator worker runs in the stack.

## Acceptance / Hardening
- [ ] Add e2e test that emits synthetic provider errors to trigger an incident and verifies alert dispatch (mock channel).
- [ ] Validate metrics cardinality in staging; adjust labels/sampling if necessary.
- [ ] Load test evaluator window rotation to ensure Redis/DB overhead stays bounded.
