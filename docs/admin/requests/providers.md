# Providers
Query adapter registrations, health probes, and incident history via `/admin/providers`.

## Prepare environment
Authenticate before polling provider telemetry.

```bash
export GATEWAY_ADMIN_BASE_URL="http://localhost:8090"
export ADMIN_EMAIL="admin@example.com"
export ADMIN_PASSWORD="change-me"
TOKEN=$(curl -sS "$GATEWAY_ADMIN_BASE_URL/admin/auth/login" -H "Content-Type: application/json" -d '{"email":"'$ADMIN_EMAIL'","password":"'$ADMIN_PASSWORD'"}' | jq -r '.access_token')
```

## List providers
Fetch each registered adapter, capabilities, and routing health.

```bash
curl -sS "$GATEWAY_ADMIN_BASE_URL/admin/providers" \
  -H "Authorization: Bearer $TOKEN" | jq '.data[] | {slug, capabilities, healthy: .health.status}'
```

## Inspect incidents
Query `/admin/providers/{slug}/incidents` (if exposed) or filter the base list for incident metadata attached to each deployment.

```bash
curl -sS "$GATEWAY_ADMIN_BASE_URL/admin/providers/openai" \
  -H "Authorization: Bearer $TOKEN" | jq '{slug, incidents: .incidents[0:3], last_probe}'
```

## Adjust routing weights
PATCH provider metadata or catalog entries to down-weight degraded regions before the auto-healer reacts.

```bash
curl -sS -X PATCH "$GATEWAY_ADMIN_BASE_URL/admin/providers/openai" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"routing_weight": 0.3}' | jq '{slug, routing_weight}'
```

## Correlate with headers
When tenants escalate incidents, capture their `X-Provider-Request-ID` and `X-Request-ID` headers to search for the matching probe in `/admin/providers`.
