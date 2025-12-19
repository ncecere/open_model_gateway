# Model A/B Testing & Shadow Traffic

## Summary
Operators need to trial new provider deployments without exposing tenants to regressions. We will add first-class experiment tooling that splits traffic across primary/variant routes and optionally mirrors (“shadow”) real requests to alternate providers while discarding their responses.

## Implementation Plan

### Routing Enhancements
1. **Experiment Metadata** – extend catalog/route records with optional `experiments` blocks (e.g., `{id: "gemini-flash-beta", bucket: 0.1, mode: "active|shadow"}`) and store assignment state per API key/tenant.
2. **Assignment Engine** – deterministic hashing (API key + experiment ID) to ensure users stay in the same bucket across calls.
3. **Shadow Execution** – when mode=`shadow`, execute the variant asynchronously, log metrics, but always return the primary response.
4. **Fallback Rules** – if the variant errors beyond a threshold, auto-disable the experiment and alert ops.

### Telemetry & Reporting
- Record experiment tags in usage/telemetry tables (`experiment_id`, `bucket`, `mode`, `primary/variant stats`).
- Extend admin Usage page with comparison charts (latency, success rate, cost) per experiment.
- Provide `/admin/experiments` API to list, enable/disable, and fetch metrics for automation.

### Configuration & UX
- Admin portal “Experiments” tab to define buckets, select target aliases, and view live health.
- CLI/YAML support so operators can seed experiments via config repos.

## Example Workflow
1. SRE defines `experiment gemini-v3` with 10% traffic, active mode.
2. Selected tenants/API keys automatically receive variant responses; remaining 90% stays on the primary alias.
3. After a week, ops compares metrics, then increases bucket to 50% or ends the experiment and promotes the variant to primary.
4. For risky upgrades, use shadow mode to gather metrics without returning variant responses.

## Components Required
- **Database**: `experiments` table + per-key assignment cache.
- **Routing Engine**: experiment-aware route selection, shadow execution hooks, failure monitoring.
- **Usage Service**: experiment metadata ingestion and aggregation.
- **Admin UI**: configuration form, status badges, charts.
- **Docs**: operator guide describing setup, API endpoints, and best practices.

## Risks & Mitigations
- *State explosion*: limit concurrent experiments per alias and enforce TTLs.
- *Cost impact*: shadow requests double provider spend; add per-experiment budget caps and warning alerts.
- *Latency*: ensure shadow calls never block primaries; use async workers.

## Next Steps
1. Design experiment schema + API.
2. Implement assignment + routing changes.
3. Wire metrics + dashboards.
4. Ship admin UI/documentation.
