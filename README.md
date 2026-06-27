# gemini-web-cli

A Go command-line interface for [Google Gemini](https://gemini.google.com) via browser cookies.

Built on the reverse-engineering work of [Gemini-API](https://github.com/HanaokaYuzu/Gemini-API) by [@HanaokaYuzu](https://github.com/HanaokaYuzu). Huge thanks for figuring out the Gemini web protocol, streaming format, and deep research flow.

## Install

```bash
go install github.com/Leechael/gemini-web-cli@latest
```

Or build from source:

```bash
git clone https://github.com/Leechael/gemini-web-cli
cd gemini-web-cli
make build    # outputs to ./bin/gemini-web-cli
```

## Quick Start

```bash
# Import cookies from browser (copy raw cookie string from DevTools)
gemini-web-cli import '_ga=GA1.1...; __Secure-1PSID=g.a000...; SID=abc...'

# Or set the path via environment variable
export GEMINI_WEB_COOKIES_JSON_PATH=cookies.json

# Ask a question
gemini-web-cli ask "What is the capital of France?"

# Continue the conversation
gemini-web-cli reply c_abc123 "And what's its population?"

# List your chats
gemini-web-cli list

# Start the OpenAI-compatible REST server
gemini-web-cli serve --state-dir ~/.local/share/gemini-web-cli/serve
```

## REST API server

`gemini-web-cli serve` starts a local OpenAI-compatible API server.

```bash
gemini-web-cli serve --port 8080 --state-dir ~/.local/share/gemini-web-cli/serve
```

Exposed endpoints:

- `GET /v1/models`
- `POST /v1/chat/completions`
- `POST /v1/research`
- `GET /v1/research/{id}`
- `GET /v1/research/{id}/status`
- `GET /v1/research/{id}/result`
- `GET /docs`
- `GET /openapi.json`

`chat_id` is a gemini-web-cli extension, not part of the OpenAI Chat Completions API. Standard OpenAI clients can omit it. When `--state-dir` is set, the server persists a chat mapping at `<state-dir>/chat-map.pb` and uses OpenAI `messages` history hashes to try to continue the matching Gemini chat automatically. If no mapping matches, the server creates a new Gemini chat and sends a flattened text prompt.

`--state-dir` also participates in cookie lookup. Serve-mode cookie priority is:

1. `--cookies-json`
2. `<state-dir>/cookies.json`
3. `$GEMINI_WEB_COOKIES_JSON_PATH`
4. Auto-discovered `cookies.json` paths
5. `GEMINI_SECURE_1PSID` / `GEMINI_SECURE_1PSIDTS`

The startup banner prints the actual cookie source and chat mapping path. Existing cookies are not migrated into `--state-dir` automatically.

Chat mapping entries are either verified or synthetic. Verified entries correspond to Gemini states produced by this server. Synthetic entries are best-effort anchors inferred from client-provided history. Forked conversation branches are not officially supported, and Gemini Web may reject or reinterpret old turn metadata. Image parts, tool calls, and function calls are not supported by the REST chat endpoint.

## MCP server

`gemini-web-cli serve` also exposes a [Model Context Protocol](https://modelcontextprotocol.io) server at `/mcp` using the Streamable HTTP transport (stateless). It lets MCP-compatible clients (Cursor, VS Code, Claude Desktop, etc.) drive Gemini deep research and one-shot prompts as tools.

```bash
gemini-web-cli serve --port 8080 --mcp-default-model gemini-3.5-flash
```

The MCP endpoint reuses the same server process, cookies, and state as the OpenAI-compatible REST API, and is reachable at `http://127.0.0.1:8080/mcp`. It is **not** behind the `--api-key` middleware, so keep it bound to `127.0.0.1` (the default) and do not expose it on a public host without your own auth proxy.

`--mcp-default-model` sets the default model for MCP tool calls that omit `model`. A per-call `model` argument overrides it; when both are empty, Gemini auto-selects (`unspecified`).

Exposed MCP tools:

- `gemini_research_create` — submit a deep research task; returns `id`, `title`, `eta_text`, `steps`. Args: `prompt` (required), `model` (optional).
- `gemini_research_status` — poll task state (`done`, `running`, `pending_confirm`, `not_research`, `empty`). Args: `id` (required).
- `gemini_research_result` — fetch the completed report text and source citations. Args: `id` (required).
- `gemini_research_list` — list completed deep research reports from the library. Args: `count` (optional, default `13`), `cursor` (optional).
- `gemini_research_reply` — send a follow-up prompt to an existing deep research chat to refine or continue the research; returns an immediate acknowledgement, then poll `gemini_research_status`. Args: `id` (required), `prompt` (required), `model` (optional).
- `gemini_ask` — single-turn prompt (search-like, no conversation state); returns `text` plus any generated image/video/media URLs. Args: `prompt` (required), `model` (optional).
- `gemini_list_models` — list available model names and display names. No args.

Deep research is asynchronous: call `gemini_research_create`, poll `gemini_research_status` until `state` is `done`, then call `gemini_research_result`. Use `gemini_research_list` to browse completed reports and `gemini_research_reply` to refine an in-progress or completed research thread.

### Client configuration

Cursor (`.cursor/mcp.json`) — supports local Streamable HTTP directly:

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

VS Code (`.vscode/mcp.json`):

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

Claude Desktop's local `claude_desktop_config.json` only launches stdio servers; to reach this local HTTP endpoint, bridge it with [`mcp-remote`](https://www.npmjs.com/package/mcp-remote):

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

Restart the client after editing its config. Keep `gemini-web-cli serve` running while you use the tools.

## Commands

### import

Parse a raw cookie string from browser DevTools and save as a JSON file.

```bash
# Save to default cookies.json (or $GEMINI_WEB_COOKIES_JSON_PATH)
gemini-web-cli import '_ga=GA1.1.123; __Secure-1PSID=g.a000...; SID=abc...'

# Save to a specific path
gemini-web-cli import '_ga=GA1.1.123; __Secure-1PSID=g.a000...' -o path/to/cookies.json
```

### ask

Single-turn question with streaming output. Supports text, image generation, video generation, and music generation.

```bash
gemini-web-cli ask "Explain quantum computing"
gemini-web-cli ask --no-stream "What is 2+2?"
gemini-web-cli --model gemini-3-flash ask "Draw a sunset"

# Generate video/music (explicit mode)
gemini-web-cli ask --mode video "A cat walking in slow motion"
gemini-web-cli ask --mode music "A short jazz melody"
gemini-web-cli ask --mode image-to-video -f photo.jpg "Animate this photo"

# Attach files
gemini-web-cli ask -f image.png "What's in this image?"
gemini-web-cli ask -f a.pdf -f b.pdf "Compare these documents"
```

The `--mode` flag controls generation type: `auto` (default), `text`, `video`, `image-to-video`, `music`.

Output includes the response text, any generated images/videos/media, and the chat ID for follow-up.

### reply

Continue an existing conversation. The chat ID comes from `ask` or `list` output.

```bash
gemini-web-cli reply c_abc123 "Tell me more"
gemini-web-cli reply --no-stream c_abc123 "Summarize"
gemini-web-cli reply --mode video c_abc123 "Now generate a video of that scene"
```

Supports the same `--mode` flag as `ask`.

### list

List chat history with pagination.

```bash
gemini-web-cli list
gemini-web-cli list --cursor <cursor>
```

### get

Get a conversation's messages, including generated images, videos, and media.

```bash
gemini-web-cli get c_abc123
gemini-web-cli get c_abc123 --max-turns 10
gemini-web-cli get c_abc123 --output chat.txt
```

Example output:

```
--- user #1 r_abc123 (2026-05-26 12:34 CST) ---
Draw a cat

--- agent #1 r_def456 (2026-05-26 12:34 CST) ---
Here's a generated cat image!

[Generated Image 1] https://lh3.googleusercontent.com/...

--- user #2 r_ghi789 (2026-05-26 12:35 CST) ---
Generate a video of a cat walking

--- agent #2 r_jkl012 (2026-05-26 12:35 CST) ---
Your video is ready!

[Generated Video 1] https://contribution.usercontent.google.com/download?...
  Thumbnail: https://lh3.googleusercontent.com/...

--- user #3 r_mno345 (2026-05-26 12:36 CST) ---
Generate a jazz melody

--- agent #3 r_pqr678 (2026-05-26 12:36 CST) ---
Here's a jazz melody for you.

[Generated Media 1] MP3: https://contribution.usercontent.google.com/download?...
[Generated Media 1] MP4: https://contribution.usercontent.google.com/download?...
[Generated Media 1] VTT: https://contribution.usercontent.google.com/download?...
```

Each turn shows `user` and `agent` blocks with request IDs and local timestamps. Generated media item numbers correspond to the `download` command's index selector.

### download

Download generated images, videos, or media by direct URL or chat ID.

```bash
# Direct URL
gemini-web-cli download "https://lh3.googleusercontent.com/..." -o image.png

# All media from a chat (images, videos, music)
gemini-web-cli download c_abc123 -o output.png
# Saves output_1.png, output_2.mp4, output_3.mp3, ...

# Specific item by index (matches get output numbering)
gemini-web-cli download c_abc123 2 -o video.mp4

# Direct URL with polling (for in-progress video/music generation)
gemini-web-cli download --poll "https://contribution.usercontent.google.com/download?..."
```

Videos and music downloads automatically poll (retry on HTTP 206) when downloading from a chat ID.

### progress

Check generation progress for deep research, video, or music tasks.

```bash
gemini-web-cli progress c_abc123
```

Output examples:

```
  Type: deep research
  Status: running
```

```
  Type: deep research
  Status: done
  Report length: 42318 chars

  Use 'report c_abc123' to retrieve the full result.
```

```
  Type: video generation
  Status: ready
  Video 1: https://contribution.usercontent.google.com/download?...

  Use 'download c_abc123' to save.
```

```
  Type: music generation
  Status: ready
  Media 1 MP3: https://contribution.usercontent.google.com/download?...
  Media 1 MP4: https://contribution.usercontent.google.com/download?...
  Media 1 VTT: https://contribution.usercontent.google.com/download?...

  Use 'download c_abc123' to save.
```

### research

Submit and manage deep research tasks.

```bash
# Submit a task
gemini-web-cli research run "Compare Rust and Go for systems programming"

# List completed reports from your library
gemini-web-cli research list
gemini-web-cli research list --count 20 --cursor <cursor>
gemini-web-cli research list --json
```

### report

Get the deep research result.

```bash
gemini-web-cli report c_abc123
gemini-web-cli report c_abc123 --output report.md
```

### chat

Chat metadata and inspection utilities.

```bash
# Show metadata for a chat
gemini-web-cli chat meta c_abc123
gemini-web-cli chat meta c_abc123 --json

# Fetch a single conversation turn by request ID
gemini-web-cli chat turn c_abc123 <requestId>
gemini-web-cli chat turn c_abc123 <requestId> --json
```

### expand-prompt

Expand a media prompt into alternative descriptions (useful for image, video, or music generation).

```bash
gemini-web-cli expand-prompt "A sunset over the ocean"
gemini-web-cli expand-prompt "A sunset over the ocean" --json
```

### models

List available models.

```bash
gemini-web-cli models
```

```
Available models for --model:
  unspecified (default)
  gemini-3.1-flash-lite (Gemini 3.1 Flash-Lite)
  gemini-3.5-flash (Gemini 3.5 Flash)
  gemini-3.1-pro [advanced] (Gemini 3.1 Pro)
  gemini-3-pro (Gemini 3 Pro)
  gemini-3-flash (Gemini 3 Flash)
  gemini-3-flash-thinking (Gemini 3 Flash Thinking)
  gemini-3-pro-plus [advanced] (Gemini 3 Pro Plus)
  gemini-3-flash-plus [advanced] (Gemini 3 Flash Plus)
  gemini-3-flash-thinking-plus [advanced] (Gemini 3 Flash Thinking Plus)
  gemini-3-pro-advanced [advanced] (Gemini 3 Pro Advanced)
  gemini-3-flash-advanced [advanced] (Gemini 3 Flash Advanced)
  gemini-3-flash-thinking-advanced [advanced] (Gemini 3 Flash Thinking Advanced)

Note: dynamic models come from the current Gemini account when cookies are available.
Use 'unspecified' to let Gemini auto-select.
```

### status

Check login status and account diagnostics.

```bash
# Full account probe (network)
gemini-web-cli status

# Cookie-only check (no network) — requires --cookies-json
gemini-web-cli status --cookies-only --cookies-json cookies.json
```

### debug

Low-level utilities for exercising RPCs directly. Useful for verifying protocol changes.

```bash
# Call any batchexecute RPC and print raw response
gemini-web-cli debug rpc otAQ7b
gemini-web-cli debug rpc MaZiqc --payload '[13,null,[1,null,1]]'
gemini-web-cli debug rpc hNvQHb --payload '["c_abc",10]' --source-cid c_abc
gemini-web-cli debug rpc cYRIkd --payload '["en"]' --pretty

# Trigger a housekeeping RPC by name
gemini-web-cli debug housekeeping heartbeat
gemini-web-cli debug housekeeping list-gems --pretty
```

Supported housekeeping names: `heartbeat`, `ui-heartbeat`, `set-lang`, `ma-gu-ac`, `list-gems`, `bulk-log`, `log-event`, `log-model-select`.

## Global Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--cookies-json` | Path to cookie JSON file | `$GEMINI_WEB_COOKIES_JSON_PATH` |
| `--model` | Model name (see `models` command) | `unspecified` |
| `--proxy` | HTTP/SOCKS proxy URL | `$HTTPS_PROXY` |
| `--account-index` | Google account index (for multi-login, e.g. `/u/2`) | — |
| `--verbose` | Debug logging to stderr | `false` |
| `--no-persist` | Don't write updated cookies back to file | `false` |
| `--request-timeout` | HTTP timeout in seconds | `300` |

## Cookies

The cookie JSON file is resolved in order:

1. `--cookies-json` flag (explicit)
2. `$GEMINI_WEB_COOKIES_JSON_PATH` environment variable
3. `./cookies.json` (project-level)
4. `~/.config/gemini-web-cli/cookies.json` (user-level)
5. `/etc/gemini-web-cli/cookies.json` (system-level)

The file accepts multiple formats:

- `{"cookies": {"name": "value", ...}}` (output of `import` command)
- `{"name": "value", ...}` (flat key-value)
- `[{"name": "...", "value": "...", ...}, ...]` (browser extension export)
- `{"cookies": [{"name": "...", "value": "...", ...}, ...]}`

The required cookie is `__Secure-1PSID`. `__Secure-1PSIDTS` is recommended but optional.

## E2E Tests

```bash
./scripts/e2e-test.sh cookies.json
```

Covers all commands with a live Gemini account.

## Acknowledgments

This project is a Go reimplementation of the HTTP-level protocol reverse-engineered by [Gemini-API](https://github.com/HanaokaYuzu/Gemini-API). The Python library by [@HanaokaYuzu](https://github.com/HanaokaYuzu) was the sole reference for understanding Gemini's streaming format, RPC protocol, cookie rotation, and deep research workflow.

## License

Same as [Gemini-API](https://github.com/HanaokaYuzu/Gemini-API).
