# Models endpoint
Use `/v1/models` to confirm which aliases are live for your tenant.

## Prepare environment
Reuse the shared variables so curl can find the router.

```bash
export GATEWAY_BASE_URL="http://localhost:8090/v1"
export OPENAI_API_KEY="sk-demo.my-secret"
```

## List models
Request the catalog and highlight provider health before sending traffic.

```bash
curl -sS "$GATEWAY_BASE_URL/models" \
  -H "Authorization: Bearer $OPENAI_API_KEY" | \
  jq '.data[] | {id, provider, healthy: .provider_health, tenant_assignable}'
```

## Filter by provider
Use `jq` or `rg` to isolate aliases that point at a specific provider or deployment region.

```bash
curl -sS "$GATEWAY_BASE_URL/models" \
  -H "Authorization: Bearer $OPENAI_API_KEY" | \
  jq '.data[] | select(.provider=="openai") | {id, region: .metadata.region}'
```

## Inspect metadata
Most entries include `metadata` describing routing weights, pricing cents, and special capabilities such as `audio_streaming` or `image_variation`.

```bash
curl -sS "$GATEWAY_BASE_URL/models/gpt-oss-20b" \
  -H "Authorization: Bearer $OPENAI_API_KEY" | jq '{id, metadata, routing}'
```

## Monitor headers
Every response returns `X-Budget-*`, `X-RateLimit-*`, `X-Provider-Request-ID`, and `X-Request-ID`; log them while validating tenant settings.
