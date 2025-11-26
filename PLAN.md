# Provider Telemetry & Alerting Plan

Objective: deliver structured provider telemetry with configurable SLIs, alerting, and admin visibility so operators can detect upstream degradation early and act before tenant impact.

## Outcomes
- Providers emit OTEL/Prometheus metrics for latency, tokens, and upstream errors; dashboards and alerts leverage the same signals.
- SLIs (latency, error rate, saturation/timeout rate) are configurable per provider entry; recent windows are persisted and queryable.
- Alert pipeline reuses existing email/webhook channels with dedupe/cooldowns and incident history.
- Admin UI surfaces live health charts and incidents; router can optionally consume health to bias failover.

## Scope (v1)
- Instrument provider adapters (sync + streaming) to emit structured metrics and spans with provider/model labels.
- SLI evaluator computing rolling windows (Redis) with persistence of incidents (Postgres).
- Alert triggers on threshold breach with cooldowns and dedupe; attaches sample errors where available.
- Read-only admin panels for health and incidents; OTEL/Prometheus endpoints export new metrics.
- Configurable defaults/env keys under `DEFAULT_PROVIDER_TELEMETRY_*`; per-entry overrides via provider registry metadata.

## Non-goals (for now)
- Automatic rerouting/weight adjustments beyond reading health flags.
- Long-term retention/warehouse exports of raw telemetry.
- Provider-specific deep diagnostics beyond upstream error/status codes.

## Design Notes
- Instrumentation: wrap provider capability interfaces so every call records timers, upstream status, token counts, and failure reasons; ensure streaming helper emits final metrics once streams close/abort.
- Metrics surface: publish OTEL metrics (histograms for latency, counters for errors/timeouts/tokens) and mirror critical gauges to Prometheus with provider/model labels.
- SLI config: define per-provider SLI profiles (p95 latency, error rate %, timeout rate) with defaults; store overrides in config/provider registry. Persist windowed aggregates in Redis keyed by provider/model + window.
- Evaluator: periodic worker computes SLIs from Redis windows; writes incident rows (opened/resolved) in Postgres with timestamps, window size, and sample errors.
- Alerting: reuse existing email/webhook channels with per-channel cooldown; dedupe by provider/model/incident type; include links to Grafana/Prom dashboards when configured.
- Router integration: expose a health/telemetry service that the existing health monitor can read; optionally annotate routing decisions with current health state for audit logs.
- Admin UI: new "Provider Health" view showing live metrics (latency/error charts), incident list, and SLI thresholds; leverage React Query + existing OTEL/Prom proxy if available.
- Configuration: env/yaml knobs for defaults, window sizes, evaluator frequency, alert cooldown, and enable/disable per provider.
- Testing: unit tests for instrumentation hooks, evaluator math, and alert dedupe; integration tests with fake providers to emit metrics and trigger incidents.

## Milestones
- M0: Define config schema + metrics naming; add data structures for SLIs and incidents.
- M1: Instrument all provider adapters and streaming helper; expose OTEL/Prom metrics.
- M2: Implement SLI evaluator + Redis windows + Postgres incidents; wire alert channels.
- M3: API surfaces for health/incidents + router consumption hook; seed sample data in dev.
- M4: Admin UI for health/incidents; docs + Grafana dashboards; harden tests/load checks.

## Dependencies & Assumptions
- Redis available for rolling windows; Postgres migrations managed via Goose.
- Existing alert channels (email/webhook) reusable; OTEL/Prom exporters already enabled.
- Provider registry metadata can accept additional telemetry configs without breaking existing adapters.

## Risks
- High-cardinality metrics if labels are too granular; mitigate with provider/model scoping and sampling.
- Streaming calls may double-count tokens if instrumentation is misplaced; ensure single close-path.
- Alert fatigue if thresholds are too sensitive; ship safe defaults and cooldowns.

## Open Questions (resolved)
- Router down-weighting: down-weight unhealthy provider instances when there are multiple entries for the same model alias (e.g., gpt-5 east-1 vs west-3). Keep single-instance aliases advisory only.
- Prometheus proxy: no proxy needed; metrics stay in Prometheus/OTEL and operators will consume via Grafana. Admin UI can rely on summaries exposed by the admin API (derived from the same metrics) without proxying Prometheus directly.
- Incident retention: 30 days by default, configurable via config file and adjustable in admin settings.
