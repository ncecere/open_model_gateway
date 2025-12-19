# Plugin & Tool Execution

## Summary
Allow tenants to register custom tools (HTTP endpoints, MCP servers, internal APIs) that the gateway can invoke in response to model tool calls/function calls. This mirrors OpenAI’s function calling but lets tenants host their own tools with managed auth, logging, and guardrails.

## Implementation Plan

### Tool Registry
- `tools` table keyed by tenant with fields: name, description, json_schema, endpoint URL, auth headers (API key, OAuth), timeout/retry policy.
- Support two kinds:
  1. **HTTP tools** – simple POST requests with JSON payloads.
  2. **MCP tools** – integrate with the Model Context Protocol (launch MCP client, call commands).

### Request Lifecycle
1. Chat/Responses handler detects tool usage (`tool_calls` from providers).
2. Router verifies the tool is registered + enabled for the tenant/model.
3. Invoke tool with structured payload (input arguments + context like tenant/model/user ID).
4. Capture response (success or failure) and re-inject it into the model conversation, just like OpenAI function calling.
5. Log invocation metadata (latency, status, arguments) for auditing/billing.

### Security & Guardrails
- Tenant-scoped secrets stored in Vault/DB (encrypted).
- Enforce timeouts, retries, and error thresholds per tool.
- Allow guardrail policies to approve/deny tool outputs.

### Admin/User Portal
- UI to register/edit tools, upload JSON schema, test invocations.
- Usage dashboards showing tool frequency, error rates, average latency.

## Example Workflow
1. Tenant registers an HTTP tool `fetch_customer_profile` with schema `{customer_id: string}`.
2. Model prompt triggers a tool call; router POSTs to the tool endpoint with `{customer_id: "123", context: {...}}`.
3. Tool responds with JSON; router formats it as model tool output and continues the conversation.

## Components Needed
- Tool service (`internal/services/tools`) + APIs (`/user/tools`, `/admin/tenants/:id/tools`).
- Executor/tool runner supporting HTTP + MCP clients.
- Request pipelines to detect tool calls and orchestrate invocation.
- Logging/observability to track tool performance.

## Risks
- Long-running or unstable tools → enforce strict timeouts and fallback to textual responses when tools fail.
- Security exposure → sanitize inputs, restrict outbound connections, and require explicit opt-in per model/tenant.

## Next Steps
1. Define tool schema + APIs.
2. Implement tool runner + integration with chat/responses pipeline.
3. Build portal UI + docs.
4. Add telemetry + billing hooks for tool usage.
