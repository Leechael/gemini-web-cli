# gemini-web-cli

A Go command-line interface for [Google Gemini](https://gemini.google.com) via browser cookies.

Built on the reverse-engineering work of [Gemini-API](https://github.com/HanaokaYuzu/Gemini-API) by [@HanaokaYuzu](https://github.com/HanaokaYuzu). Huge thanks for figuring out the Gemini web protocol, streaming format, and deep research flow.

## Install

```bash
go install github.com/AIO-Starter/gemini-web-cli@latest
```

Or build from source:

```bash
git clone https://github.com/AIO-Starter/gemini-web-cli
cd gemini-web-cli
make build    # outputs to ./bin/gemini-web-cli
```

## Quick Start

```bash
# Set up your cookies file (see Cookies section below)
export COOKIES="--cookies-json cookies.json"

# Ask a question
gemini-web-cli $COOKIES ask "What is the capital of France?"

# Continue the conversation
gemini-web-cli $COOKIES reply c_abc123 "And what's its population?"

# List your chats
gemini-web-cli $COOKIES list
```

## Commands

### ask

Single-turn question with streaming output.

```bash
gemini-web-cli --cookies-json cookies.json ask "Explain quantum computing"
gemini-web-cli --cookies-json cookies.json ask --no-stream "What is 2+2?"
gemini-web-cli --cookies-json cookies.json --model gemini-2.0-flash ask "Draw a sunset"
```

Output includes the response text, any generated images, and the chat ID for follow-up.

### reply

Continue an existing conversation. The chat ID comes from `ask` or `list` output.

```bash
gemini-web-cli --cookies-json cookies.json reply c_abc123 "Tell me more"
gemini-web-cli --cookies-json cookies.json reply --no-stream c_abc123 "Summarize"
```

### list

List chat history with pagination.

```bash
gemini-web-cli --cookies-json cookies.json list
gemini-web-cli --cookies-json cookies.json list --cursor <cursor>
```

### read

Read a conversation's messages.

```bash
gemini-web-cli --cookies-json cookies.json read c_abc123
gemini-web-cli --cookies-json cookies.json read c_abc123 --max-turns 10
gemini-web-cli --cookies-json cookies.json read c_abc123 --output chat.txt
```

Images in the conversation are shown with global numbering:

```
--- message 1 ---
[User] Draw a cat
[Generated Image 1] https://lh3.googleusercontent.com/...

--- message 2 ---
[User] Now draw a dog
[Generated Image 2] https://lh3.googleusercontent.com/...
```

The image numbers correspond to the `download` command's index selector.

### download

Download generated images by direct URL or chat ID.

```bash
# Direct URL
gemini-web-cli --cookies-json cookies.json download "https://lh3.googleusercontent.com/..." -o image.png

# All images from a chat
gemini-web-cli --cookies-json cookies.json download c_abc123 -o images.png
# Saves images_1.png, images_2.png, ...

# Specific image by index (matches read output numbering)
gemini-web-cli --cookies-json cookies.json download c_abc123 2 -o second.png
```

### research

Deep research workflow: submit a topic, check progress, retrieve the report.

```bash
# Submit
gemini-web-cli --cookies-json cookies.json research send --prompt "Compare Rust and Go for systems programming"

# Check progress
gemini-web-cli --cookies-json cookies.json research check c_abc123

# Get result
gemini-web-cli --cookies-json cookies.json research get c_abc123
gemini-web-cli --cookies-json cookies.json research get c_abc123 --output report.md
```

### models

List available models.

```bash
gemini-web-cli models
```

```
Available models for --model:
  unspecified (default)
  gemini-2.0-flash
  gemini-2.5-pro [advanced]
  gemini-2.5-flash [advanced]
  gemini-3.1-pro [advanced]
  gemini-3.0-flash
```

### inspect

Diagnose cookie and account status.

```bash
# Cookie-only check (no network)
gemini-web-cli --cookies-json cookies.json inspect --cookies-only

# Full account probe
gemini-web-cli --cookies-json cookies.json inspect
```

## Global Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--cookies-json` | Path to cookie JSON file | — |
| `--model` | Model name (see `models` command) | `unspecified` |
| `--proxy` | HTTP/SOCKS proxy URL | `$HTTPS_PROXY` |
| `--account-index` | Google account index (for multi-login, e.g. `/u/2`) | — |
| `--verbose` | Debug logging to stderr | `false` |
| `--no-persist` | Don't write updated cookies back to file | `false` |
| `--request-timeout` | HTTP timeout in seconds | `300` |

## Cookies

This CLI authenticates using browser cookies exported from [gemini.google.com](https://gemini.google.com). The `--cookies-json` flag accepts multiple formats:

- `{"name": "value", ...}` (flat key-value)
- `{"cookies": {"name": "value", ...}}`
- `{"cookies": [{"name": "...", "value": "...", ...}, ...]}`
- `[{"name": "...", "value": "...", ...}, ...]` (browser export format)

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
