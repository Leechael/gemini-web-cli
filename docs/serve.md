# Serve

`gemini-web-cli serve` starts one process with:

- OpenAI-compatible REST at `/v1/` — [http-api.md](http-api.md)
- MCP at `/mcp` — [mcp.md](mcp.md)
- Swagger UI at `/docs`

```bash
gemini-web-cli serve [flags]
```

Docker defaults differ (listen `0.0.0.0`, cookies `/cookies`, state `/state`): [docker.md](docker.md).

## Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--port` | Port to listen on (or `GEMINI_WEB_CLI_PORT`) | `8080` |
| `--host` | Host to bind to (or `GEMINI_WEB_CLI_HOST`) | `127.0.0.1` |
| `--api-key` | API key for `/v1` endpoints (or `GEMINI_WEB_CLI_API_KEY`) | — |
| `--expose-thoughts` | Include model thoughts/reasoning in API responses (or `GEMINI_WEB_CLI_EXPOSE_THOUGHTS=1`) | `false` |
| `--state-dir` | Directory for state (cookie lookup + chat map persistence; or `GEMINI_WEB_CLI_STATE_DIR`) | — |
| `--mcp-default-model` | Default model for MCP tool calls that omit `model` | — |
| `--verbose` | Debug logging to stderr (or `GEMINI_WEB_CLI_VERBOSE=1`) | `false` |
| `--rpc-log` | Enable RPC request/response logging (or `GEMINI_WEB_CLI_RPC_LOG=1`; dir via `GEMINI_WEB_CLI_RPC_LOG_DIR`) | `false` |

The startup banner prints the cookie source, listen URLs, and chat mapping path.

## RPC logs

`--rpc-log` / `GEMINI_WEB_CLI_RPC_LOG=1` records outbound Gemini requests and responses. Off by default. `GEMINI_WEB_CLI_RPC_LOG_DIR` changes the directory (default `data/rpc_logs`; Docker: `/state/rpc_logs`) but does not enable logging.

Layout: daily NDJSON index `<log-dir>/YYYY-MM-DD.ndjson`, bodies under `<log-dir>/blobs/YYYY-MM-DD/`. Cookie, Authorization, Set-Cookie, batchexecute `at`, init access-token, and init session-ID values are redacted. Prompts, uploads, and model responses remain readable. Files older than seven days are removed on start and rotation.

## Cookie priority

1. `--cookies-json` flag
2. `<state-dir>/cookies.json`
3. `$GEMINI_WEB_COOKIES_JSON_PATH`
4. Auto-discovered `cookies.json` paths
5. `GEMINI_SECURE_1PSID` / `GEMINI_SECURE_1PSIDTS`

Existing cookies are not copied into `--state-dir`.

## Multiple accounts

`serve` can use several Google accounts to spread rate limits. New chats, research tasks, and notebooks go round-robin; on failure (e.g. HTTP 429) the next account is tried. Follow-ups for a chat/research/notebook id stay on the owning account (recorded at creation, re-probed after restart). Streaming failovers only before the first delta.

```bash
gemini-web-cli import '<cookies_alice>' -o accounts/alice.json
gemini-web-cli import '<cookies_bob>' -o accounts/bob.json

gemini-web-cli serve --cookies-json accounts/
# equivalent: --cookies-json accounts/alice.json --cookies-json accounts/bob.json
```

`$GEMINI_WEB_COOKIES_JSON_PATH` accepts a path list (`:` on Linux/macOS) or a directory.

One-shot commands (`ask`, `reply`, …) and `status --cookies-only` use the first resolved cookie file. `gemini_research_list` merges reports from every account, newest first; cross-account pagination cursors are not supported.
