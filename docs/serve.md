# Serve Mode

`gemini-web-cli serve` starts a local HTTP server that exposes an OpenAI-compatible REST API, a Model Context Protocol (MCP) endpoint, and Swagger docs — all from a single process sharing the same cookies and state.

## Starting the server

```bash
gemini-web-cli serve [flags]
```

The following flags are available:

| Flag | Description | Default |
|------|-------------|---------|
| `--port` | Port to listen on | `8080` |
| `--host` | Host to bind to | `127.0.0.1` |
| `--api-key` | API key for `/v1` endpoints (or `GEMINI_WEB_CLI_API_KEY`) | — |
| `--expose-thoughts` | Include model thoughts/reasoning in API responses (or `GEMINI_WEB_CLI_EXPOSE_THOUGHTS=1`) | `false` |
| `--state-dir` | Directory for state (cookie lookup + chat map persistence) | — |
| `--mcp-default-model` | Default model for MCP tool calls that omit `model` | — |
| `--rpc-log` | Enable RPC request/response logging to `data/rpc_logs` (or `GEMINI_WEB_CLI_RPC_LOG_DIR`) | `false` |

The startup banner prints the active cookie source, LAN IPs, and chat mapping path.

## REST API

### Endpoints

The following endpoints are exposed:

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/models` | List available models |
| `POST` | `/v1/chat/completions` | OpenAI-compatible chat completions |
| `POST` | `/v1/research` | Submit a deep research task |
| `GET` | `/v1/research/{id}` | Get research summary |
| `GET` | `/v1/research/{id}/status` | Poll research state |
| `GET` | `/v1/research/{id}/result` | Fetch completed report |
| `GET` | `/docs` | Swagger UI |
| `GET` | `/openapi.json` | OpenAPI spec |

When `--api-key` is set, all `/v1/` endpoints require `Authorization: Bearer <key>` or `X-API-Key: <key>`.

### Chat completions

The `POST /v1/chat/completions` endpoint is OpenAI-compatible. Standard clients work without modification. One extension: `chat_id` in the response carries the underlying Gemini chat ID.

Image parts, tool calls, and function calls are not supported.

### Chat state mapping

When `--state-dir` is set, the server persists a chat mapping at `<state-dir>/chat-map.pb`. The server hashes the OpenAI `messages` history to find matching Gemini chats and auto-continues them. If no match is found, a new Gemini chat is created with a flattened text prompt.

Chat mapping entries are **verified** (produced by this server) or **synthetic** (inferred from client-provided history). Forked conversation branches are not officially supported.

### Cookie priority in serve mode

Cookie files are resolved in the following order:

1. `--cookies-json` flag
2. `<state-dir>/cookies.json`
3. `$GEMINI_WEB_COOKIES_JSON_PATH`
4. Auto-discovered `cookies.json` paths
5. `GEMINI_SECURE_1PSID` / `GEMINI_SECURE_1PSIDTS`

Existing cookies are not migrated into `--state-dir` automatically.

## MCP server

The MCP endpoint is at `http://127.0.0.1:<port>/mcp` (Streamable HTTP, stateless). It is **not** protected by `--api-key` — keep it bound to `127.0.0.1` and do not expose it publicly without your own auth proxy.

`--mcp-default-model` sets the default model for tool calls that omit `model`. If neither the flag nor a per-call `model` is set, Gemini auto-selects.

### Tools

The following tools are available:

| Tool | Description |
|------|-------------|
| `gemini_ask` | Single-turn prompt; returns `text` plus any generated image/video/media URLs. Args: `prompt` (required), `model` (optional). |
| `gemini_research_create` | Submit a deep research task; returns `id`, `title`, `eta_text`, `steps`. Args: `prompt` (required), `model` (optional). |
| `gemini_research_status` | Poll task state (`done`, `running`, `pending_confirm`, `not_research`, `empty`). Args: `id` (required). |
| `gemini_research_result` | Fetch the completed report text and source citations. Args: `id` (required). |
| `gemini_research_list` | List completed deep research reports from the library. Args: `count` (optional, default `13`), `cursor` (optional). |
| `gemini_research_reply` | Send a follow-up prompt to an existing research chat; poll `gemini_research_status` after. Args: `id` (required), `prompt` (required), `model` (optional). |
| `gemini_list_models` | List available model names and display names. No args. |

Deep research flow: call `gemini_research_create` → poll `gemini_research_status` until `state` is `done` → call `gemini_research_result`. Use `gemini_research_reply` to refine a completed thread.

## Client configuration

### Cursor

`.cursor/mcp.json` — supports Streamable HTTP directly:

```json
{
  "mcpServers": {
    "gemini-web-cli": {
      "type": "streamable-http",
      "url": "http://127.0.0.1:8080/mcp"
    }
  }
}
```

### VS Code

`.vscode/mcp.json`:

```json
{
  "servers": {
    "gemini-web-cli": {
      "type": "http",
      "url": "http://127.0.0.1:8080/mcp"
    }
  }
}
```

### Claude Desktop

`claude_desktop_config.json` — only launches stdio servers; bridge with [`mcp-remote`](https://www.npmjs.com/package/mcp-remote):

```json
{
  "mcpServers": {
    "gemini-web-cli": {
      "command": "npx",
      "args": ["-y", "mcp-remote", "http://127.0.0.1:8080/mcp"]
    }
  }
}
```

Restart the client after editing its config. Keep `gemini-web-cli serve` running while the tools are in use.
