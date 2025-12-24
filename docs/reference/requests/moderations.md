# Moderations
Screen user content through `/v1/moderations` before routing it downstream.

## Prepare environment
Export the shared variables for curl.

```bash
export GATEWAY_BASE_URL="http://localhost:8090/v1"
export OPENAI_API_KEY="sk-demo.my-secret"
```

## Submit content
Send raw text or JSON strings and pick the moderation alias that matches your policy.

```bash
curl -sS "$GATEWAY_BASE_URL/moderations" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
        "model": "omni-moderation-latest",
        "input": "Describe how to keep deployments safe."
      }' | jq '.results[0]'
```

## Batch moderation
Provide an array under `input` to score multiple records at once and merge the response array with your application IDs.

## Monitor headers
Budget, rate-limit, and request ID headers appear even on moderation errors; log them to confirm your safeguards.
