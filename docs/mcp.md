# MCP

`gemini-web-cli serve` exposes a Model Context Protocol endpoint at `/mcp` (Streamable HTTP, stateless).

It is **not** protected by `--api-key`. Keep it on `127.0.0.1`, or put your own auth proxy in front.

`--mcp-default-model` sets the default model for tool calls that omit `model`. If neither that nor a per-call `model` is set, Gemini auto-selects.

## Tools

| Tool | Description |
|------|-------------|
| `gemini_ask` | Single-turn prompt; returns `text` plus any generated image/video/media URLs. Args: `prompt` (required), `model` (optional), `notebook` (optional, scopes the new chat to a notebook). |
| `gemini_research_create` | Submit a deep research task; returns `id`, `title`, `eta_text`, `steps`. Args: `prompt` (required), `model` (optional). |
| `gemini_research_status` | Poll task state (`done`, `running`, `pending_confirm`, `not_research`, `empty`). Args: `id` (required). |
| `gemini_research_result` | Fetch the completed report text and source citations. Args: `id` (required). |
| `gemini_research_list` | List completed deep research reports from the library. Args: `count` (optional, default `13`), `cursor` (optional). |
| `gemini_research_reply` | Follow-up on an existing research chat; poll `gemini_research_status` after. Args: `id` (required), `prompt` (required), `model` (optional). |
| `gemini_list_models` | List available model names and display names. No args. |
| `gemini_notebook_create` | Create a notebook; returns `resource` (`notebooks/<uuid>`) and `title`. Args: `title` (required). |
| `gemini_notebook_get` | Notebook title, emoji, and source list. Args: `id` (required). |
| `gemini_notebook_list_chats` | List chats inside a notebook, newest first. Args: `id` (required). |
| `gemini_notebook_add_file_source` | Upload a local file (on the serve host) and attach it. Args: `id` (required), `path` (required). |
| `gemini_notebook_add_url_source` | Attach a web URL as a source. Args: `id` (required), `url` (required). |
| `gemini_notebook_remove_source` | Remove a source. Args: `source` (required, `notebooks/<uuid>/sources/<sid>`). |

Deep research: `gemini_research_create` → poll `gemini_research_status` until `done` → `gemini_research_result`. Refine with `gemini_research_reply`.

## Client configuration

Keep `serve` running while the client uses the tools. Restart the client after editing config.

### Codex

`~/.codex/config.toml` (or project `.codex/config.toml`):

```toml
[mcp_servers.gemini-web-cli]
url = "http://127.0.0.1:8080/mcp"
```

Then `codex mcp list`. In a session, `/mcp` shows the connected server.

### Cursor

`.cursor/mcp.json`:

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

`claude_desktop_config.json` only launches stdio servers. Bridge with [`mcp-remote`](https://www.npmjs.com/package/mcp-remote):

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

Docker: if you published `8080:8080`, the URL from the host is still `http://127.0.0.1:8080/mcp`.
