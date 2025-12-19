# Observability Dashboards

## Summary
Bundle ready-to-use Grafana dashboards and lightweight portal charts so operators can visualize budgets, provider health, latency, and usage hotspots without stitching together custom queries.

## Implementation Overview

### Metrics Pipeline
1. **OTEL → Prometheus** reference config (docker-compose + Terraform snippets) bundling the existing OTEL exporters.
2. **Dashboard Library** – JSON dashboards for Grafana covering:
   - Provider latency/error rates (per provider/model).
   - Tenant budgets/spend with burn rate projections.
   - Usage heatmaps (requests/tokens by alias, top tenants).
   - Batch/job throughput and failure reasons.

### Portal Visualizations
1. Embed mini dashboards inside the admin portal using existing Usage APIs (Chart.js or Recharts components):
   - Budgets meter with historic trend.
   - Provider health spark-lines from the telemetry service.
   - Model leaderboard (latency vs cost).
2. Add download links so ops can import the Grafana JSON with one click.

### Configuration
- Provide `docs/observability/metrics.md` updates with step-by-step instructions for hooking Prometheus/Grafana to OTEL/Redis/Postgres.
- Support optional alert rules (Grafana Alerting) tied to the telemetry SLIs so dashboards double as alert surfaces.

## Example Dashboard Set
- **Ops Overview**: p50/p95 latency, error rates, circuit breaker statuses, active incidents.
- **Finance View**: spend per tenant/model, forecasting, usage anomalies.
- **Batch Monitor**: in-progress job counts, success/failure breakdown, throughput.

## Components Needed
- Dashboard JSON assets committed under `deploy/grafana/`.
- CLI/script to load dashboards (e.g., `scripts/load-dashboards.sh`).
- Portal components + API glue for inline charts.
- Documentation + Terraform examples.

## Risks
- Dashboard drift if metrics change → add CI check that validates dashboard JSON against sample Prom metrics.
- Multi-tenant data exposure → portal charts must enforce tenant roles and only show aggregate views where appropriate.

## Next Steps
1. Inventory existing OTEL metrics; add any missing series (budget, provider error counts).
2. Build dashboards in Grafana locally, export JSON, and version control them.
3. Implement portal chart components + docs.
4. Publish deployment guide + Terraform snippets.
