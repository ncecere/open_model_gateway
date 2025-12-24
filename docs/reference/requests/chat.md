# Chat and responses
Send chat completions or Responses API payloads with SSE support when needed.

## Prepare environment
Export the same base URL and API key before running these flows.

```bash
export GATEWAY_BASE_URL="http://localhost:8090/v1"
export OPENAI_API_KEY="sk-demo.my-secret"
```

## Send chat completions
POST message arrays with JSON payloads and match the `model` to an allowed alias.

```bash
curl -sS "$GATEWAY_BASE_URL/chat/completions" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
        "model": "gpt-oss-20b",
        "messages": [
          {"role": "system", "content": "You route LLM traffic for us."},
          {"role": "user", "content": "Summarize how failover works."}
        ]
      }' | jq '.choices[0].message.content'
```

## Stream chat completions
Request SSE with `Accept: text/event-stream` and keep the connection open using `curl -sN`.

```bash
curl -sN "$GATEWAY_BASE_URL/chat/completions" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
        "model": "gpt-oss-20b",
        "stream": true,
        "messages": [
          {"role": "user", "content": "Send an incremental update."}
        ]
      }'
```

## Use multimodal inputs
Upload files via `/v1/files` and reference them in messages with `{"type":"image_file","image_file":{"file_id":"file-abc"}}` blocks.

## Call the Responses API
Submit structured `input` arrays (text, images, tools) for parity with OpenAI's `/v1/responses` surface.

```bash
curl -sS "$GATEWAY_BASE_URL/responses" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
        "model": "gpt-4o-mini",
        "input": [
          {"role": "system", "content": "You write operational updates."},
          {"role": "user", "content": "Summarize usage metering."}
        ]
      }' | jq '.output[0]'
```

## Stream Responses API
Set `stream:true` or send the `stream` top-level sibling to mirror OpenAI SSE semantics.

```bash
curl -sN "$GATEWAY_BASE_URL/responses" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
        "model": "gpt-4o-mini",
        "stream": true,
        "input": [
          {"role": "user", "content": "Send partial chunks."}
        ]
      }'
```

## Monitor headers
Capture `X-Budget-*`, `X-RateLimit-*`, `X-Provider-Request-ID`, and `X-Request-ID` for troubleshooting and usage audits.
