# Tenant Guardrails & Policy Engine Roadmap

## Summary
Provide per-tenant policy enforcement so every request/response passes through configurable guardrails: moderation, PII redaction, regex filters, or custom webhooks.

## Implementation Overview

1. **Policy Definition**
   - Schema for guardrail configs (enabled categories, actions: block, redact, warn) stored per tenant/API key.
   - Support multiple stages: pre-request (input), post-response (output).

2. **Execution Pipeline**
   - Middleware intercepts `/v1/*` calls, runs configured checks, and decides whether to continue, redact, or reject.
   - Integrate moderation providers (OpenAI, custom) and allow custom HTTP callbacks for complex logic.

3. **Management UI**
   - Admin portal page to configure policies, preview effects, and view violation history.

## Usage Examples

### Enforcing PII Guardrails
```bash
curl -X PATCH https://router.example.com/admin/tenants/{tenant_id}/guardrails \
  -H "Authorization: Bearer sk-admin" \
  -H "Content-Type: application/json" \
  -d '{
        "input": {
          "moderation_model": "omni-moderation-latest",
          "pii_detection": true,
          "action": "block"
        },
        "output": {
          "pii_redact": true,
          "disclaimer": "This answer was sanitized."
        }
      }'
```

### Custom Webhook Guardrail
```bash
curl -X PATCH https://router.example.com/admin/tenants/{tenant_id}/guardrails \
  ...
  -d '{
        "output": {
          "webhook": {
            "url": "https://policy.corp/api/validate",
            "timeout_ms": 1500,
            "action_on_fail": "block"
          }
        }
      }'
```

## Implementation Details

| Area | Notes |
|------|-------|
| Policy storage | `tenant_guardrails` table with JSON configs for input/output stages. |
| Execution | Chain of interceptors in public API router; short-circuit on block. |
| Logging | Record violations with reason, snippet hashes, tenant, API key. |
| Alerts | Optionally send guardrail violations via existing alert channels. |

## Next Steps
1. Finalize policy schema + defaults.
2. Implement middleware + provider integrations.
3. Build admin UI for configuration & monitoring.
4. Document API + best practices for tenants.
