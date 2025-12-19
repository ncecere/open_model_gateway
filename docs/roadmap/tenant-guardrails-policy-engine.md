# Tenant Guardrails & Policy Engine

## Summary
Give administrators tenant-specific safety controls so every request/response passes through configurable guardrails (moderation, regex filters, data masking). Policies must be enforced automatically, logged for auditing, and manageable via admin UI + API.

## Implementation Plan

### Policy Model
- `guardrail_policies` table keyed by tenant (or API key override) storing:
  - Pre-request rules: moderation providers, regex/keyword blocks, max prompt length.
  - Post-response rules: moderation categories, PII detectors, custom webhook validators.
  - Actions per rule: `block`, `warn`, `redact`, `inject_disclaimer`.
- Allow multiple layers: tenant default, API key override, model-specific exceptions.

### Enforcement Pipeline
1. **Pre-request**
   - Run moderation provider (OpenAI, Azure) + regex checks before routing.
   - If rule triggers `block`, short-circuit with structured error.
   - Support “warn” actions that annotate responses (e.g., `guardrail_warnings`).
2. **Post-response**
   - Re-run moderation/regex on the generated content.
   - Redact text per configured masks (replace with `***`).
   - Inject disclaimers when required.
   - Optional webhook call for custom validation (tenant-provided endpoint).
3. **Logging**
   - Record every rule evaluation in `guardrail_events` (tenant, key, rule, action, timestamp) for auditing + dashboards.

### Admin UI & API
- Admin Settings → Guardrails tab with:
  - Policy builder (choose provider, thresholds, actions).
  - Rule test harness (paste prompt/response, see rule outcomes).
  - Event log table with filters.
- API endpoints: `GET/PUT /admin/tenants/:id/guardrails`, `GET /admin/guardrails/events`.

## Example Policy
```yaml
policies:
  - tenant_id: tenant-123
    pre_request:
      - type: moderation
        provider: openai
        action: block
      - type: regex
        pattern: "(?i)credit card"
        action: block
    post_response:
      - type: pii_redact
        fields: ["ssn", "email"]
      - type: disclaimer
        text: "This response may contain sensitive data."
```

## Components Needed
- Policy storage + evaluation engine (`internal/guardrails`).
- Integration hooks in HTTP handlers and executor pipeline.
- Event logging + analytics (extend usage service or new service).
- Admin portal UI + API docs.

## Risks & Mitigations
- Performance impact → run guardrails asynchronously where possible, cache moderation results, batch webhook calls.
- False positives → provide per-rule overrides and whitelists; add “test mode” before enforcement.

## Next Steps
1. Design policy schema + evaluation engine.
2. Add enforcement hooks in request/response pipeline.
3. Build admin UI + event logging.
4. Document configuration + best practices.
