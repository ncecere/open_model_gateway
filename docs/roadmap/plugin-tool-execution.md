# Plugin & Tool Execution Roadmap

## Summary

Enable tenants to register custom tools (HTTP endpoints, serverless functions, or managed scripts) that large-language-model requests can invoke mid-conversation. This provides first-class support for Enterprise RAG, workflow automation, and web-search augmentation without forcing tenants to build bespoke orchestration layers.

## Implementation Overview

1. **Tool Registry**
   - Create `tools` tables/services storing tenant-scoped metadata: `id`, `name`, description, JSON schema, and invocation config.
   - Support two invocation types:
     - `http`: direct webhooks/serverless endpoints with headers, retries, secrets.
     - `mcp`: reference to a tenant MCP server (`url`, credentials, `tool_name`) so the gateway syncs schemas automatically.
   - Admin APIs (`/admin/tools`) for CRUD operations, validation, and tests.

2. **Model/Route Configuration**
   - Extend model catalog entries with `allowed_tools` arrays and per-tool quotas.
   - Admin UI toggles to attach/detach tools per deployment.

3. **Runtime Orchestration**
   - Public `/v1/chat/completions` (and future `/v1/responses`) request context includes available tools.
   - When the provider (or gateway) emits a tool call, the router validates inputs, executes the HTTP call (with retries/timeouts), and logs results. Partial failures can either bubble up (error tool call) or allow fallback responses.

4. **Security & Guardrails**
   - Integration with tenant guardrail policies: limit which tools users can invoke, redact sensitive arguments, and throttle misuse.
   - Secrets fetched from Vault/ENV so tool headers aren’t stored in plaintext.

5. **Observability**
   - Usage logging: request ID, tool ID, latency, status, payload sizes.
   - Metrics for success/failure rates, enabling SLOs and alerts.

## Usage Examples

### 1. Enterprise RAG (Vector Search)
- **Tool**: `search_docs` hits `https://rag.backend.corp/api/search` with JSON payload `{"query":"...","top_k":5}`.
- **Workflow**: User asks “Summarize our SOC2 policy updates.” The LLM triggers `search_docs`, receives three snippets, and crafts a grounded answer.
- **Benefit**: The tenant keeps data in their private vector store while giving the model self-service retrieval.
- **Example**:
  ```bash
  curl https://router.example.com/v1/chat/completions \
    -H "Authorization: Bearer sk-tenant-user" \
    -H "Content-Type: application/json" \
    -d '{
          "model": "gpt-4o-enterprise",
          "messages": [
            {"role":"system","content":"Answer using internal docs only."},
            {"role":"user","content":"Summarize our SOC2 policy updates for 2025."}
          ],
          "tools": [
            {
              "type": "function",
              "function": {
                "name": "search_docs",
                "description": "Enterprise doc search",
                "parameters": {
                  "type": "object",
                  "properties": {
                    "query": {"type":"string"},
                    "top_k": {"type":"integer","default":5}
                  },
                  "required": ["query"]
                }
              }
            }
          ],
          "tool_choice": "auto"
        }'
  ```

### 2. Web Search Augmentation
- **Tool**: `web_search` calls a serverless function that wraps SerpAPI/Brave Search.
- **Workflow**: Prompt “What’s the latest news on Anthropic?” The LLM detects stale knowledge, calls `web_search`, and summarizes the returned articles.
- **Benefit**: Keeps answers current without hardcoding search logic in the client, and allows per-tenant search providers or caching policies.
- **Example**:
  ```bash
  curl https://router.example.com/v1/chat/completions \
    -H "Authorization: Bearer sk-tenant-user" \
    -H "Content-Type: application/json" \
    -d '{
          "model": "gpt-4o-enterprise",
          "messages": [
            {"role":"user","content":"What is the latest news on Anthropic?"}
          ],
          "tools": [
            {
              "type": "function",
              "function": {
                "name": "web_search",
                "description": "Fetch current articles",
                "parameters": {
                  "type": "object",
                  "properties": {
                    "query": {"type":"string"},
                    "freshness": {"type":"string","enum":["1d","7d","30d"],"default":"1d"}
                  },
                  "required": ["query"]
                }
              }
            }
          ],
          "tool_choice": "auto"
        }'
  ```

### 3. Workflow Automation
- **Tool**: `create_ticket` posts to the tenant’s ServiceNow/Jira webhook.
- **Workflow**: A support assistant notices an outage mention and automatically files an incident by calling `create_ticket`, returning the ticket ID to the user.
- **Benefit**: LLMs can trigger downstream actions safely, with full audit logs and rate limits managed by the gateway.
- **Example**:
  ```bash
  curl https://router.example.com/v1/chat/completions \
    -H "Authorization: Bearer sk-tenant-user" \
    -H "Content-Type: application/json" \
    -d '{
          "model": "gpt-4o-ops",
          "messages": [
            {"role":"user","content":"Users in us-east report 500 errors. Open an incident."}
          ],
          "tools": [
            {
              "type": "function",
              "function": {
                "name": "create_ticket",
                "description": "Create a ServiceNow incident",
                "parameters": {
                  "type": "object",
                  "properties": {
                    "summary": {"type":"string"},
                    "priority": {"type":"string","enum":["P1","P2","P3"],"default":"P2"}
                  },
                  "required": ["summary"]
                }
              }
            }
          ],
          "tool_choice": "auto"
        }'
  ```

## Implementation Details

| Area | Notes |
|------|-------|
| **Data model** | `tools` table linked to `tenants`; join table `model_allowed_tools`. Include JSON schema blobs and encrypted secrets references. |
| **Execution engine** | Reuse existing HTTP client pool with configurable retries/timeouts. Add circuit breakers per tool to prevent cascading failures. |
| **Public API** | No client changes beyond existing OpenAI tool schema; gateway injects tool metadata automatically or validates incoming `tools` arrays. |
| **Error handling** | Map HTTP errors/timeouts to tool call failures and optionally let the model recover (return `tool_error`). Provide deterministic error codes for auditing. |
| **Billing** | Record tool invocation counts/latency for future chargeback or usage limits. |
| **Security** | Allow per-tool auth options (static header, OAuth client credentials, signed presigned URLs). Support VPC peering/private endpoints for low-latency internal calls. |

## Next Steps

1. Design the `tools` schema + admin APIs.
2. Extend model catalog service and UI to attach tools.
3. Implement runtime orchestration with logging/metrics.
4. Dogfood with a built-in `web_search` tool to validate UX.
### 4. MCP-backed Web Search
- **Tool**: `mcp_web_search` references the tenant’s MCP server; schema is fetched automatically.
- **Workflow**: Same OpenAI-style tool-calling payload, but execution happens via MCP `callTool`.
- **Example Registration**:
  ```bash
  curl -X POST https://router.example.com/admin/tools \
    -H "Authorization: Bearer sk-admin" \
    -H "Content-Type: application/json" \
    -d '{
          "tenant_id": "tenant-123",
          "id": "mcp_web_search",
          "name": "Web Search (MCP)",
          "invoke": {
            "type": "mcp",
            "server": {
              "url": "https://mcp.tenant.corp",
              "client_id": "gateway",
              "client_secret": "${MCP_SECRET}"
            },
            "tool_name": "web_search"
          }
        }'
  ```
- **Client Request** (standard OpenAI format):
  ```bash
  curl https://router.example.com/v1/chat/completions \
    -H "Authorization: Bearer sk-tenant-user" \
    -H "Content-Type: application/json" \
    -d '{
          "model": "gpt-4o-enterprise",
          "messages": [
            {"role": "user", "content": "What is the latest news on Anthropic?"}
          ],
          "tools": [
            {
              "type": "function",
              "function": {
                "name": "mcp_web_search",
                "description": "Fetch current web results",
                "parameters": {
                  "type": "object",
                  "properties": {
                    "query": {"type": "string"},
                    "freshness": {"type": "string", "enum": ["1d","7d","30d"], "default": "1d"}
                  },
                  "required": ["query"]
                }
              }
            }
          ],
          "tool_choice": "auto"
        }'
  ```
