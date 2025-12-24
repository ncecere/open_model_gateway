# Troubleshooting Reference

Use this checklist to resolve common issues across admins, tenant owners, and standard users. Capture the request ID (`X-Request-ID`) and tenant/key identifiers before escalating so operators can trace logs and usage rows quickly.

## Error Matrix

| Error | Meaning | User Action | Tenant Owner Action | Admin Action |
| --- | --- | --- | --- | --- |
| `401 unauthorized` | Key revoked, expired, or copied incorrectly | Request a new key; confirm correct tenant | Verify the key still exists; reissue if rotated | Check audit logs for revocations; rotate compromised credentials |
| `402 budget_exceeded` | Budget exhausted for key or tenant | Pause workload until refresh | Increase per-key budget (if approved) or delay jobs | Adjust tenant/global budgets or refresh schedule; notify finance |
| `404 model_not_found` | Alias not assigned to tenant | Ask owner to enable alias | Enable alias under **Models → Assignments** | Confirm alias exists in catalog and is tenant-assignable |
| `409 conflict` | Duplicate idempotency key | Change client idempotency tokens | Ensure client uses unique IDs per request | Investigate Redis idempotency cache for leaks |
| `429 rate_limit_exceeded` | RPM/TPM/parallel cap hit | Add retry/backoff with `Retry-After` header | Raise per-key limits if needed | Raise tenant defaults or global settings if capacity allows |
| `provider_unavailable` | Upstream unhealthy and no fallback available | Retry later or switch models | Provide fallback aliases; communicate ETA | Adjust routing weights, disable bad providers, track via `/admin/providers` |
| `500` (generic) | Unexpected backend error | Retry with exponential backoff | Provide request IDs to admins | Inspect router logs, traces, and request payloads |

## Diagnostics Checklist

1. **Confirm tenant + key** – mismatched keys often cause errors. Check the UI or `/admin/tenants/{id}/api-keys`.
2. **Inspect headers** – `X-RateLimit-*` and `X-Budget-*` explain most throttling issues.
3. **Review usage dashboards** – sudden spikes indicate runaway jobs. Use `Usage → Keys` to isolate.
4. **Check provider incidents** – admins can open **Providers** or hit `/admin/providers` to see upstream health.
5. **Look at OTEL/Prometheus** – traces (`X-Request-ID`) and metrics quickly reveal saturated queues or retries.
6. **Reproduce with curl** – run the relevant script from `Code_Examples/curl` to rule out SDK/client bugs.

## Escalation Paths

- **User → Tenant Owner**: include tenant name, key prefix, endpoint, request payload summary, timestamps, and headers.
- **Tenant Owner → Admin**: include request IDs, impact (blocked users, latency), and any mitigations already attempted.
- **Admin → Provider**: include `X-Provider-Request-ID`, timestamps, region, and payload class (chat, embeddings, etc.). Log details in `agents.md` or your incident tracker.

## Maintenance Reminders

- Update this file whenever new error codes or workflows are introduced.
- Cross-link the relevant guides (admin/tenant/user) when you add deeper context.
- Reference `docs/runtime/observability.md` for enabling debug logs and traces during investigations.
