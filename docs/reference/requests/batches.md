# Batches
Submit asynchronous workloads through `/v1/batches` and monitor their lifecycle.

## Prepare environment
Export the shared variables before uploading NDJSON payloads.

```bash
export GATEWAY_BASE_URL="http://localhost:8090/v1"
export OPENAI_API_KEY="sk-demo.my-secret"
```

## Upload input files
Stage NDJSON payloads via `/v1/files` with `purpose=batch` and note the returned `file_id`.

```bash
curl -sS "$GATEWAY_BASE_URL/files" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -F "purpose=batch" \
  -F "file=@Code_Examples/data/batch_chat.ndjson" | jq '.id'
```

## Create a batch
Reference the uploaded file, target endpoint, and completion window.

```bash
curl -sS "$GATEWAY_BASE_URL/batches" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
        "input_file_id": "file-abc123",
        "endpoint": "/v1/chat/completions",
        "completion_window": "24h",
        "metadata": {"source": "smoke-test"}
      }' | jq '{id, status, request_counts}'
```

## Poll batch status
GET `/v1/batches/{id}` or list batches with pagination to observe worker progress and request tallies.

```bash
curl -sS "$GATEWAY_BASE_URL/batches/batch_123" \
  -H "Authorization: Bearer $OPENAI_API_KEY" | jq '{status, request_counts, output_file_id, error_file_id}'
```

## Download outputs
Fetch `output_file_id` or `error_file_id` via `/v1/files/{id}/content` to inspect NDJSON results.

## Monitor headers
Budget and rate-limit headers cover the batch management requests while batch execution costs accrue per completed item.
