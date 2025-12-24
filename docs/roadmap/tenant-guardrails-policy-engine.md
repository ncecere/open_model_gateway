# Tenant Guardrails & Policy Engine

## Summary
Give administrators and tenants a unified way to enforce safety policies (moderation + PII redaction) before and after every LLM call. Policies must:
- Be authorable via config, API, or UI.
- Support multiple enforcement stages (pre-request, post-response, tool output).
- Run against open-weight models such as **gpt-oss-safeguard** (20B/120B) for policy-based moderation.
- Redact PII using the Go `github.com/aavaz-ai/pii-scrubber` library and custom regex detectors.
- Apply flexibly per tenant, per model alias, or both, with full audit logging.

## Architecture & Components

### Policy Model
- `guardrail_policies` table stores reusable policy definitions:
  - `type`: `moderation_model`, `pii_redaction`, `regex`, `webhook`, etc.
  - `stage`: `pre_request`, `post_response`, `tool_output`.
  - `config`: JSON blob (moderation policy text, model choice, PII detectors, actions).
  - `action`: `block`, `warn`, `redact`, `log_only`, `inject_disclaimer`.
- `guardrail_assignments` table links policies to tenants, models, or API keys (hierarchical precedence: global → tenant → model → key).

### Moderation via gpt-oss-safeguard
- Config fields:
  - `model_variant`: `gpt-oss-safeguard-20b` or `gpt-oss-safeguard-120b`.
  - `policy_text`: developer-authored policy.
  - `threshold`/severity mapping → action.
- Runtime:
  1. Stage filter collects applicable moderation policies.
  2. Sends `{policy_text, content}` to our hosted gpt-oss-safeguard inference endpoint.
  3. Parses verdict + reasoning; triggers action (block/warn/etc.).
- Hosting: run both variants inside our inference stack; policy decides which to call. Expose routing metadata so ops can change default per tenant/model.

### PII Redaction via `pii-scrubber`
- Policy config:
  - `detectors`: label + preset from `pii.StandardPIIData` or custom regex definition.
  - `strategy`: `mask_all`, `mask_last4`, `replace_with_token`, `block`.
  - `scope`: prompt, completion, tool output.
- Runtime: policy engine wraps the configured text and passes it to `pii-scrubber` (optionally augmented with custom regex/NLP). Matching spans are redacted or trigger blocks per policy.

### Enforcement Pipeline
1. **Policy selection**: gather policies by scope (global + tenant + model + key) and stage.
2. **Execution order**:
   - Pre-request: moderation → regex → PII.
   - Post-response: moderation → PII → disclaimers/webhooks.
3. **Actions**:
   - `block`: return structured error.
   - `warn`: include `guardrail_warnings` metadata.
   - `redact`: mutate payload before returning to caller.
   - `inject_disclaimer`: append text to completion.
4. **Logging**: `guardrail_events` table with tenant, policy ID, decision, severity, action, and optional chain-of-thought.

### Management Surfaces
- **Config**: YAML/Helm seeds (policies + assignments) loaded at startup.
- **API**:
  - `GET/POST /admin/guardrails/policies`
  - `GET/POST /admin/guardrails/assignments`
  - Tenant-scoped versions (`/user/guardrails/...`) where allowed.
  - `GET /admin/guardrails/events` for audit.
- **UI**:
  - Admin portal Guardrails tab: create/edit policies, choose gpt-oss-safeguard variant, stage, detectors, actions, and target tenants/models.
  - Test harness: run sample prompt/response against policies before publishing.
  - Event log viewer with filters.

## Sample Policy Configuration
```yaml
policies:
  - id: default-moderation
    type: moderation_model
    stage: pre_request
    action: block
    config:
      model_variant: gpt-oss-safeguard-20b
      policy_text: |
        Block hate speech, extremism, self-harm instructions...

  - id: finance-pii
    type: pii_redaction
    stage: post_response
    action: redact
    config:
      detectors:
        - label: credit_card
          preset: CREDIT_CARD
          strategy: mask_all_but_last4
        - label: email
          preset: EMAIL
          strategy: replace_with("[EMAIL REDACTED]")

assignments:
  - tenant_id: org-123
    policy_ids: [default-moderation, finance-pii]

  - model_alias: creative-lab
    policy_ids: [default-moderation]
```

## Implementation Roadmap

### Phase 1 – Foundations
- [ ] Define DB schema (`guardrail_policies`, `guardrail_assignments`, `guardrail_events`).
- [ ] Create policy CRUD API + admin UI scaffolding.
- [ ] Implement policy evaluation engine interface (stages, actions, logging hooks).

### Phase 2 – Moderation Engine
- [ ] Host gpt-oss-safeguard 20B/120B models (LM Studio/vLLM) behind internal inference API.
- [ ] Build Go client with retries, timeouts, and circuit breakers.
- [ ] Add moderation policy executor (pre/post stages) + chain-of-thought logging (configurable).

### Phase 3 – PII Redaction
- [ ] Integrate `github.com/aavaz-ai/pii-scrubber` with customizable detector lists and redaction strategies.
- [ ] Support custom regex detectors defined in policy config.
- [ ] Implement redaction actions (mask, replace, block) and ensure idempotent logging.

### Phase 4 – Assignment & UI/Docs
- [ ] Implement hierarchical assignment resolution (global → tenant → model → key).
- [ ] Admin UI for policy editor, test harness, and assignments.
- [ ] Update user/tenant portal where self-service guardrails are permitted.
- [ ] Publish documentation: config format, API samples, guardrail best practices.

## Risks & Mitigations
- **Latency/cost**: gpt-oss-safeguard is compute-heavy. Mitigate with caching, asynchronous evaluation (post-response), lightweight pre-filters.
- **False positives**: allow dry-run/test mode, severity thresholds, and per-rule overrides.
- **Secrets leakage**: store policy + detector configs encrypted; restrict access to guardrail events containing sensitive context.

## Deliverables Checklist
- [ ] Policy storage + evaluation engine (`internal/guardrails`).
- [ ] gpt-oss-safeguard inference service + moderation executor.
- [ ] PII redaction executor using `pii-scrubber`.
- [ ] Integration hooks in HTTP handlers/executor pipeline.
- [ ] Admin API + UI + documentation.
- [ ] Telemetry + audit dashboards for guardrail events.
> **Note:** The guardrails system is not implemented yet. This roadmap remains blocked until guardrail storage, templates, and enforcement are available.
