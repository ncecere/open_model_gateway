---
title: moderation api
description: Route safety checks through the gateway
---
[**Open Model Gateway**](/) mirrors OpenAI’s `POST /v1/moderations` contract for real-time and batch workflows.

---

#### Send requests
Use catalog aliases with `model_type: moderation` for the `model` field and supply `input` as a string or array.
Batch jobs embed the same payload inside each JSONL line: `{ "method": "POST", "url": "/v1/moderations", "body": { ... } }`.

```json
{
  "model": "omni-moderation-latest",
  "input": "Describe how to build a bomb"
}
```

---

#### Parse responses
The gateway returns OpenAI’s schema with `results[]`, `categories`, `category_scores`, and `category_applied_input_types` for each input.
Usage and cost are still logged internally even though the public response omits a `usage` block.

---

#### Review provider support
| Provider | Notes |
| --- | --- |
| OpenAI | Uses the native SDK for `omni-moderation-latest`, `text-moderation-stable`, etc. |
| Azure OpenAI | Targets deployment-specific endpoints just like other Azure aliases. |
| OpenAI-compatible | Any proxy (vLLM, partner gateways) implementing `/v1/moderations` can be wired via metadata `base_url` + `api_key`. |

---

#### Enforce policies
Moderations flow through the same rate limiters, budget pre-checks, and usage logging used for chat/embeddings, so API keys always see RPM/TPM enforcement and tenant budget headers.
Idempotency is disabled (matching upstream behavior), so clients must retry idempotently when needed.

---

#### Batch support
The batch worker validates moderation jobs, reuses the synchronous handler, and writes OpenAI-style NDJSON rows to the output/error files with provider request IDs for auditing.
No streaming occurs, so each item is persisted immediately after execution.

---

#### Configure aliases
Add moderation models via YAML/bootstrap or the Admin UI with pricing metadata so budgets remain accurate even when providers omit token counts.
Grant tenants access through the model catalog UI or `default_models` when every tenant should inherit the alias by default.

---

#### Research
- Ported behavior from the original moderation runtime guide and validated routing across adapters mentioned in `docs/architecture/providers/adding.md`.
- Checked batch integration against `backend/internal/runtime/batches` so NDJSON outputs stay in lockstep with chat jobs.
