# Guardrail Enforcement

Tenant and API-key guardrail policies now ship with the runtime. The backend
loads the effective config (`tenant` → `api-key` override) before invoking any
upstream provider and enforces the rules at multiple stages.

## Storage & APIs

- Policies live in `guardrail_policies` (scoped to tenant or API key) and can be
  managed through `/admin/tenants/:id/guardrails` or
  `/admin/tenants/:id/api-keys/:key/guardrails`.
- Each policy is a JSON blob matching `internal/guardrails.Config`. The current
  MVP supports keyword blocklists for prompts/responses plus moderation stubs
  for future providers.
- Guardrail events (including violations) land in `guardrail_events` so we can
  surface them in future dashboards/alerts.

## Enforcement Coverage (v1)

| Endpoint | Pre-check | Post-check | Notes |
| --- | --- | --- | --- |
| `/v1/chat/completions` | Entire prompt history | Full completion body | Streaming SSE chunks are buffered per choice; violations emit `event: error` with `{ "error": { "type": "guardrail_violation" } }` and terminate the stream. |
| `/v1/embeddings` | All text inputs | N/A | Currently only prompt checks – embeddings responses do not expose text to scan. |
| `/v1/images/*` | Prompt text (generate/edit/variation) | Revised prompt strings in responses | Non-textual image payloads are not inspected yet. |

When a rule blocks the prompt or response we return `403` with OpenAI-style
error content: `message` will be `prompt rejected by guardrails` or
`response blocked by guardrails`. The handler also records a guardrail event so
operators can audit the violations.

## Streaming behavior

Streaming chat uses a guardrail monitor that accumulates each choice’s emitted
text. Once a chunk matches a blocked keyword, the handler:

1. Records the violation.
2. Emits `event: error` with the guardrail payload so clients can display a
   friendly message.
3. Stops forwarding additional chunks and finalizes the usage record with a
   `403` status.

This mirrors OpenRouter’s “time to first token” semantics and preserves
idempotency logging.

## Admin UI Workflow

- **Tenants** → **Edit tenant** now includes a *Guardrail policy* card. Toggle it
  on to override the inherited defaults, edit prompt/response keyword lists, and
  optionally configure a moderation provider + action (block or warn) and
  enable/disable the policy without deleting it.
- **API keys** → select a key → dialog exposes the same guardrail controls so a
  single key can opt into stricter rules than the tenant default. Saving writes
  to `/admin/tenants/:id/api-keys/:key/guardrails`.
- All UI changes hit the same JSON schema as the raw API, so you can script the
  policies via `curl` or the admin portal interchangeably.

## Usage & Analytics

- Guardrail-blocked requests are now recorded as billable error codes. The
  admin Usage dashboard surfaces a "Guardrail blocks" KPI next to requests,
  spend, and tokens, plus per-tenant/user/model breakdown lists include counts
  so you can identify noisy actors quickly.
- Daily usage points (returned by `/admin/usage/summary` and `/admin/usage/*`)
  include `guardrail_blocks` so downstream dashboards can plot the trend.
- `/admin/guardrails/events` lists individual violations with filters for
  tenant, API key, action, stage, and time range. The admin portal exposes this
  feed under **Usage → Guardrails** with pagination and tenant filters so ops
  can quickly inspect offending requests.
- Moderation provider `webhook` is supported: set `moderation.provider="webhook"`
  along with `moderation.webhook_url` (plus optional `webhook_auth_header`,
  `webhook_auth_value`, and `timeout_seconds`). Every prompt/response is posted
  as `{ "stage": "prompt|response", "content": "..." }`, and the webhook
  returns `{ "action": "allow|warn|block", "category": "pii", "violations": [] }`.

## Alerts & Notifications

- Tenants that have alerts enabled (emails/webhooks configured under budgets)
  now receive guardrail notifications when a block occurs. The dispatcher
  reuses the existing cooldown window (`alert_cooldown_seconds`) so each tenant
  is notified at most once per interval even if multiple requests are blocked.
- Email/webhook payloads include the guardrail stage, action, and any matched
  keywords so downstream tooling (Slack, PagerDuty, etc.) can triage the issue.
- Notifications are triggered from the same runtime contexts as budget alerts,
  so you can disable guardrail alerts by turning off tenant alerts entirely or
  by clearing destination channels.

## Next Steps

- Build a guardrail events feed (API + UI) so operators can inspect the
  payloads/categories behind each block.
- Expand enforcement beyond keyword blocking (moderation webhooks, response
  redaction) and wire alerts/webhooks into the existing budget dispatcher.
