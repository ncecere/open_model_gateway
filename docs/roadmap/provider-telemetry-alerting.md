# Provider Telemetry & Alerting Roadmap

## Summary
Add first-class monitoring for upstream providers (OpenAI, Azure, Bedrock, etc.) so operators can detect latency spikes, error storms, or saturation before tenants feel impact. Alerts should route through the same email/webhook channels already used for budget events.

## Implementation Overview

1. **Metrics capture**
   - Extend provider adapters to emit structured telemetry per request: latency, upstream HTTP status, token counts, retries, failover triggers.
   - Export to OTEL/Prometheus with tags for provider, model alias, region, and tenant.

2. **Health windows & SLIs**
   - Maintain rolling windows (e.g., 1m/5m) per provider to compute error rate, p95 latency, success volume.
   - Compare against configurable thresholds (global defaults + per-provider overrides).

3. **Alerting**
   - When an SLI breaches its threshold, enqueue incidents via email/webhook (reuse budget alert infrastructure).
   - Include metadata: affected provider, metric values, suggested actions (failover, disable alias).

4. **UI & APIs**
   - Admin dashboard renders provider health cards with current status, historical spark-lines, and active incidents.
   - `/admin/providers/health` endpoint for automation.

## Usage Examples

### Detecting Azure Outage
```bash
curl https://router.example.com/admin/providers/health \
  -H "Authorization: Bearer sk-admin" | jq '.providers[] | select(.provider=="azure")'
```
- If error rate > threshold, alert fires to ops@ and Slack webhook, while the router de-routes Azure aliases.

### Monitoring Bedrock Latency via Prometheus
```bash
curl http://otel-gateway:9464/metrics | grep provider_latency_seconds_bucket | grep bedrock
```
- SRE dashboards show latency spikes, prompting scaling or failover.

## Implementation Details

| Area | Notes |
|------|-------|
| Metrics | Use OTEL Span attributes (`provider`, `alias`, `status`) and push to Prometheus/Grafana stack. |
| Storage | Redis or in-memory ring buffer to compute moving averages quickly. |
| Alert thresholds | Config file + `/admin/settings` API to set defaults and overrides. |
| Failover hooks | Tie into existing router circuit breaker to automatically disable unhealthy providers. |
| Auditing | Log every alert emission for compliance and postmortems. |

## Next Steps
1. Define telemetry schema in `internal/observability`.
2. Wire adapters to emit spans/metrics.
3. Build rolling SLI evaluator + alert dispatcher.
4. Add admin UI panels and documentation.
