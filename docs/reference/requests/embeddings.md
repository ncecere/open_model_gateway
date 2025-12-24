# Embeddings
Generate vector representations for text chunks or JSON payloads.

## Prepare environment
Reuse the base URL and key exports for consistent curl behavior.

```bash
export GATEWAY_BASE_URL="http://localhost:8090/v1"
export OPENAI_API_KEY="sk-demo.my-secret"
```

## Request embeddings
Send inputs as a string, array of strings, or token arrays depending on the target model.

```bash
curl -sS "$GATEWAY_BASE_URL/embeddings" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
        "model": "text-embedding-3-large",
        "input": [
          "Tenant isolation keeps customers separated.",
          "Failover replays requests against secondary providers."
        ]
      }' | jq '{embedding: .data[0].embedding[0:8], usage: .usage}'
```

## Tune batching
Send up to the provider's documented limit per request and monitor `usage.total_tokens` to right-size batching.

## Include metadata
Set `user` or adapter-specific metadata in the payload to help audit downstream usage.

## Monitor headers
Every embedding response includes the standard `X-Budget-*`, `X-RateLimit-*`, and tracing headers.
