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
```

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

# Generate video/music
gemini-web-cli ask "Generate a 5 second video of a cat walking"
gemini-web-cli ask "Generate a short jazz melody"

# Attach files
gemini-web-cli ask -f image.png "What's in this image?"
gemini-web-cli ask -f a.pdf -f b.pdf "Compare these documents"
```

Output includes the response text, any generated images/videos/media, and the chat ID for follow-up.

### goog

Google search via Gemini (shortcut for `ask "@Google ..."`).

```bash
gemini-web-cli goog "latest Go release notes"
gemini-web-cli goog --no-stream "weather in Tokyo"
```

### reply

Continue an existing conversation. The chat ID comes from `ask` or `list` output.

```bash
gemini-web-cli reply c_abc123 "Tell me more"
gemini-web-cli reply --no-stream c_abc123 "Summarize"
```

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
--- message 1 ---
[User] Draw a cat
[Generated Image 1] https://lh3.googleusercontent.com/...

--- message 2 ---
[User] Generate a video of a cat walking
[Gemini] Your video is ready!
[Generated Video 1] https://contribution.usercontent.google.com/download?...
  Thumbnail: https://lh3.googleusercontent.com/...

--- message 3 ---
[User] Generate a jazz melody
[Gemini] Here's a jazz melody for you.
[Generated Media 1] MP3: https://contribution.usercontent.google.com/download?...
```

The item numbers correspond to the `download` command's index selector.

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
  Type: video generation
  Status: ready
  Video 1: https://contribution.usercontent.google.com/download?...

  Use 'download c_abc123' to save.
```

```
  Type: music generation
  Status: ready
  Media 1 MP3: https://contribution.usercontent.google.com/download?...

  Use 'download c_abc123' to save.
```

### research

Submit a deep research task.

```bash
gemini-web-cli research "Compare Rust and Go for systems programming"
```

### report

Get the deep research result.

```bash
gemini-web-cli report c_abc123
gemini-web-cli report c_abc123 --output report.md
```

### models

List available models.

```bash
gemini-web-cli models
```

```
Available models for --model:
  unspecified (default)
  gemini-3-pro
  gemini-3-flash
  gemini-3-flash-thinking
  gemini-3-pro-plus [advanced]
  gemini-3-flash-plus [advanced]
  gemini-3-flash-thinking-plus [advanced]
  gemini-3-pro-advanced [advanced]
  gemini-3-flash-advanced [advanced]
  gemini-3-flash-thinking-advanced [advanced]
```

### status

Check login status and account diagnostics.

```bash
# Cookie-only check (no network)
gemini-web-cli status --cookies-only

# Full account probe
gemini-web-cli status
```

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

The cookie JSON file path is resolved in order:

1. `--cookies-json` flag
2. `$GEMINI_WEB_COOKIES_JSON_PATH` environment variable

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

Runs 35 tests covering all commands with a live Gemini account.

## Acknowledgments

This project is a Go reimplementation of the HTTP-level protocol reverse-engineered by [Gemini-API](https://github.com/HanaokaYuzu/Gemini-API). The Python library by [@HanaokaYuzu](https://github.com/HanaokaYuzu) was the sole reference for understanding Gemini's streaming format, RPC protocol, cookie rotation, and deep research workflow.

## License

Same as [Gemini-API](https://github.com/HanaokaYuzu/Gemini-API).
