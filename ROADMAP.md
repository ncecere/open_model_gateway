# Open Model Gateway Roadmap

This roadmap now focuses on open initiatives. Completed streams (telemetry & alerting, self-service API keys, Responses API, batches/files/moderations/audio/images parity, etc.) are documented separately in the CHANGELOG and release notes.

## Active Initiatives

| Initiative | Summary | Plan |
| --- | --- | --- |
| Fine-Grained Tenant RBAC | Tighten tenant roles, allow per-member budgets + tenant limits, attach/detach curated models while keeping tenant budgets and model metadata central-admin only. | `docs/roadmap/fine-grained-tenant-rbac.md` |
| Model A/B Testing & Shadow Traffic | Add experiment buckets and shadow routing so ops can evaluate new deployments before full cutover, with telemetry + UI reporting. | `docs/roadmap/model-ab-testing-shadow-traffic.md` |
| Observability Dashboards | Ship Grafana dashboards + portal charts covering budgets, provider health, usage hotspots, and provide Terraform/k8s snippets for OTEL → Prometheus → Grafana. | `docs/roadmap/observability-dashboards.md` |
| Plugin & Tool Execution | Allow tenants to register HTTP/MCP tools that the router can invoke in response to model tool calls, with logging and guardrails. | `docs/roadmap/plugin-tool-execution.md` |
| Tenant Guardrails & Policy Engine | Policy-based moderation (gpt-oss-safeguard) and PII redaction per tenant/model with pre/post stages and UI/API management. | `docs/roadmap/tenant-guardrails-policy-engine.md` |
| Provider Coverage – Google AI Studio | Add Gemini/Imagen via Google AI Studio REST endpoints, config blocks, and telemetry. | `docs/roadmap/provider-coverage-google-ai-studio.md` |
| Provider Coverage – Mistral AI | Native adapter for Mistral chat/embeddings (hosted + private endpoints) with pricing/telemetry integrations. | `docs/roadmap/provider-coverage-mistral-ai.md` |
| Provider Coverage – Cerebras | Adapter for Cerebras inference (cloud/on-prem) so private clusters can route through OMG with health/telemetry hooks. | `docs/roadmap/provider-coverage-cerebras.md` |
| Provider Coverage – Ollama | Lightweight adapter for Ollama servers (local/on-prem) to run smaller models via the gateway. | `docs/roadmap/provider-coverage-ollama.md` |
| Assistants/Threads/Runs | Long-term goal to implement OpenAI-style assistants (threads, runs, tool calling, retrieval). | _TBD_ |
| Responses Tool Parity | Extend `/v1/responses` to support tool calls and file search (matching OpenAI). | _TBD_ |

## Completed Initiatives

| Initiative | Summary | Docs |
| --- | --- | --- |
| Usage Exports & Billing Hooks | Self-serve CSV/Parquet exports and optional billing webhooks so finance teams can ingest spend data without DB access. | `docs/roadmap/usage-exports-billing-hooks.md` |
| Provider Coverage – vLLM / TGI | Generic inference-server adapter so customers can route private Llama/Qwen deployments through the gateway. | `docs/roadmap/provider-coverage-vllm.md` |

> For details on completed items, refer to the CHANGELOG and the relevant docs under `docs/roadmap/` (e.g., `provider-telemetry-alerting.md`, `self-service-api-keys.md`).
