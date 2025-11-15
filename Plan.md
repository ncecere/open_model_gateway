# Tenant Guardrails & Policy Engine Plan

## Objectives
1. Allow admins to define per-tenant (and per-API-key) guardrail policies covering prompt/response moderation, keyword filters, and escalation hooks.
2. Enforce guardrails for every `/v1/*` request without degrading latency more than a few ms.
3. Record violations for observability/audit, expose counts in UI, and optionally trigger alerts/webhooks.

## Functional Scope
- **Configuration**
  - Policy schema (JSON/YAML) with: moderation provider (OpenAI, Azure, custom), classification categories, severity handling (block, redact, warn), regex/keyword lists, request/response size limits, optional custom webhook (sync/async) for custom policy engines.
  - Inheritance: global defaults → tenant overrides → API-key overrides.
- **Execution Pipeline**
  - Pre-request stage: run keyword/regex filters, optional custom webhook, moderation call on prompt. Actions: block, redact text, attach warnings.
  - Post-response stage: run moderation on provider response, optional webhook, response redaction (messages, tool outputs), optional disclaimer injection.
  - Enforcement integrates with budget/rate-limit contexts and surfaces rejection reasons to client as OpenAI-style errors.
- **Admin UX**
  - Policy editor (form + JSON view) on tenant + API-key pages, including toggle to inherit global settings.
  - Dashboard widgets summarizing violations, top categories, last alert.
- **Telemetry & Alerts**
  - Persist guardrail events, expose `/admin/guardrails/events` for UI and optional webhook dispatch.

## Work Breakdown
1. **Schema & Storage**
   - Define `guardrail_policies` table (tenant_id, api_key_id, config JSON, versioning, timestamps).
   - Extend config loader to allow bootstrap defaults.
2. **Policy Engine Package**
   - `internal/guardrails` module exposing `Evaluator` with `PreCheck(ctx, req)` and `PostCheck(ctx, req, resp)`.
   - Providers: built-in regex/keyword filter, OpenAI moderation adapter, custom webhook executor (HTTP, configurable timeout/retries).
   - Result struct with actions (`allow`, `block`, `redact`, `warn`, `inject_disclaimer`).
3. **Pipeline Integration**
   - Hook pre-check inside public handler before provider routing; reuse existing error helpers for blocked requests.
   - Hook post-check after provider response (sync + streaming). For streaming, buffer until first chunk or apply incremental filters? MVP: collect final response before flush for guardrail enforcement (adds slight delay but simplest), mark TODO for streaming chunk redaction.
   - Option to skip guardrails for admin/system requests.
4. **Audit & Metrics**
   - Insert guardrail events into `guardrail_events` table (tenant_id, api_key_id, alias, action, category, payload hash).
   - Emit metrics counters (blocked, warned) labeled by tenant/model/category.
   - Wire optional alert notifications (reuse budget alert sink).
5. **Admin UI**
   - Add Guardrails tab on tenant + API key drawers; form controls for toggling categories, severity, regex entries.
   - Guardrail events table + charts.
6. **Docs & Examples**
   - Update ROADMAP/README with configuration snippet.

## Milestones
1. **M1 – Backend foundation**: DB tables, policy schema, evaluator stub, API endpoints for CRUD (admin).
2. **M2 – Enforcement**: integrate pre/post checks for chat/completions & embeddings; record events; basic metrics.
3. **M3 – Admin UX**: UI forms, events view, status badges.
4. **M4 – Advanced features**: custom webhooks, streaming-safe redaction, alerting.

## Next Steps (In Progress)
1. **Reusable Guardrail Policies** – add a dedicated Guardrails section in the admin sidebar where operators can define named policies (keywords, moderation provider, webhook settings, redaction flags) and reuse them across tenants, API keys, and model aliases; tenant/key/model editors will pick from these policies instead of embedding raw JSON.
2. **Response Redaction** – allow guardrails to mask or strip sensitive tokens instead of blocking whole responses; wire redaction into sync + streaming flows so SSE clients receive sanitized chunks.
3. **Streaming Safeguards** – add per-model/per-provider streaming knobs (pause/abort thresholds, per-choice enforcement) so high-risk models enforce stricter guardrails mid-stream.
4. **Additional Moderation Adapters** – plug in hosted classifiers (OpenAI Moderation, Azure Content Safety, Anthropic) alongside the new webhook option so tenants can choose managed providers without hosting their own webhook.

## Current Progress
- Plan drafted (this document);
- `.gitignore` updated to exclude Plan.md.
- Guardrail persistence, evaluator, and admin CRUD APIs implemented (`guardrail_policies`/`guardrail_events` tables, `internal/guardrails`, admin handlers).
- Guardrails enforced across chat (sync + SSE), embeddings, and all image endpoints with guardrail-aware usage logging + SSE error payloads.
- Usage logger now records guardrail violations with billable costs and distinct error codes, enabling analytics filters; runtime docs updated (`docs/runtime/guardrails.md`).
- Tenant + API key editors expose guardrail overrides in the admin portal, and the Usage dashboard tracks guardrail blocks next to spend/tokens; README + runtime docs describe the workflow.
- Guardrail events feed (`/admin/guardrails/events`) shipped with admin UI filters under Usage → Guardrails, alongside documentation updates.
- Guardrail alerts reuse tenant budget alert channels (email/webhook) with cooldown enforcement so blocks can notify ops automatically.
- Moderation webhook adapter available (config + admin UI fields) so tenants can route prompts/responses to custom policy engines; docs now explain the webhook contract.
