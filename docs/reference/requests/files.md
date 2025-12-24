# Files
Use `/v1/files` to stage assets for chat, images, batches, or assistants-style workflows.

## Prepare environment
Keep the standard exports handy for curl.

```bash
export GATEWAY_BASE_URL="http://localhost:8090/v1"
export OPENAI_API_KEY="sk-demo.my-secret"
```

## Upload a file
Send multipart form data, set the purpose, and record the returned file ID for downstream references.

```bash
curl -sS "$GATEWAY_BASE_URL/files" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -F "purpose=assistants" \
  -F "file=@Code_Examples/data/sample_knowledge.txt" | jq '{id, filename, bytes}'
```

## List files
Enumerate stored files and inspect size or status.

```bash
curl -sS "$GATEWAY_BASE_URL/files" \
  -H "Authorization: Bearer $OPENAI_API_KEY" | jq '.data[] | {id, purpose, bytes}'
```

## Download content
Append `/content` to fetch the raw bytes and redirect them to disk.

```bash
curl -sS "$GATEWAY_BASE_URL/files/file-abc123/content" \
  -H "Authorization: Bearer $OPENAI_API_KEY" --output payload.bin
```

## Delete files
Call `DELETE /v1/files/{id}` once the asset is no longer needed.

```bash
curl -sS -X DELETE "$GATEWAY_BASE_URL/files/file-abc123" \
  -H "Authorization: Bearer $OPENAI_API_KEY" | jq
```

## Monitor headers
Reuse the budget, rate-limit, and tracing headers to confirm that storage actions still respect tenant policies.
