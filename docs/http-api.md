# HTTP API

`gemini-web-cli serve` exposes an OpenAI-compatible REST API, plus research and notebooks. Interactive docs: `http://127.0.0.1:8080/docs` (OpenAPI at `/openapi.json`).

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/models` | List available models |
| `POST` | `/v1/chat/completions` | OpenAI-compatible chat completions |
| `POST` | `/v1/research` | Submit a deep research task |
| `GET` | `/v1/research/{id}` | Research summary |
| `GET` | `/v1/research/{id}/status` | Poll research state |
| `GET` | `/v1/research/{id}/result` | Completed report |
| `POST` | `/v1/notebooks` | Create a notebook |
| `GET` | `/v1/notebooks/{id}` | Get a notebook |
| `GET` | `/v1/notebooks/{id}/chats` | List notebook chats |
| `POST` | `/v1/notebooks/{id}/sources` | Add a file or URL source |
| `DELETE` | `/v1/notebooks/{id}/sources/{sid}` | Remove a source |
| `GET` | `/docs` | Swagger UI |
| `GET` | `/openapi.json` | OpenAPI spec |

When `--api-key` or `GEMINI_WEB_CLI_API_KEY` is set, `/v1/` requires `Authorization: Bearer <key>` or `X-API-Key: <key>`. `/mcp` is not covered.

## Chat completions

`POST /v1/chat/completions` is OpenAI-compatible. Standard clients work without modification. Extension: `chat_id` in the response is the Gemini chat ID.

Image parts, tool calls, and function calls are not supported.

```bash
curl http://127.0.0.1:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"model":"gemini-3.8-flash","messages":[{"role":"user","content":"hello"}]}'
```

Streaming: set `"stream": true`. Continue a chat: send `chat_id` from a previous response.

## Chat state mapping

When `--state-dir` / `GEMINI_WEB_CLI_STATE_DIR` is set, the server persists `<state-dir>/chat-map.pb`. It hashes the OpenAI `messages` history to find a matching Gemini chat and auto-continues. Otherwise it starts a new chat with a flattened text prompt.

Entries are **verified** (produced by this server) or **synthetic** (inferred from client history). Forked branches are not supported.

## Research

```bash
curl -X POST http://127.0.0.1:8080/v1/research \
  -H 'Content-Type: application/json' \
  -d '{"prompt":"Research topic here"}'

curl http://127.0.0.1:8080/v1/research/{id}
curl http://127.0.0.1:8080/v1/research/{id}/status
curl http://127.0.0.1:8080/v1/research/{id}/result
```

## Next

- [Serve flags](serve.md)
- [MCP](mcp.md)
