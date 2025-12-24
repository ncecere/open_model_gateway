# Images
Call `/v1/images/*` for generations, edits, and variations across providers.

## Prepare environment
Load the same base URL and API key exports before invoking curl.

```bash
export GATEWAY_BASE_URL="http://localhost:8090/v1"
export OPENAI_API_KEY="sk-demo.my-secret"
```

## Run generations
Send JSON prompts and decode the first `b64_json` field into a file.

```bash
curl -sS "$GATEWAY_BASE_URL/images/generations" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
        "model": "gpt-image-1-mini",
        "prompt": "A neon data center monitored by friendly operators"
      }' | jq -r '.data[0].b64_json' | base64 --decode > gateway.png
```

## Submit edits
Upload the base image and optional mask via multipart form data, then name the resulting artifact for traceability.

```bash
curl -sS "$GATEWAY_BASE_URL/images/edits" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -F "model=gpt-image-1" \
  -F "image=@gateway.png" \
  -F "mask=@mask.png" \
  -F "prompt=Replace racks with glowing cubes" | jq '.data[0].b64_json' | base64 --decode > edit.png
```

## Create variations
Pass another multipart request with `image=@path` and omit the mask to generate alternative renders.

```bash
curl -sS "$GATEWAY_BASE_URL/images/variations" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -F "model=gpt-image-1" \
  -F "image=@gateway.png" \
  -F "n=2" | jq '.data[].b64_json'
```

## Mind provider metadata
Check the model metadata for `image_edit_supported` and cost overrides (`price_image_*_cents`) because some providers only support generations.

## Monitor headers
Every image response returns the standard budget, rate limit, and tracing headers for auditing.
