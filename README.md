# gemini-web-cli

Use [Google Gemini](https://gemini.google.com) from the terminal or as an API server -- no API key needed, just your browser cookies.

[中文](README.zh.md)

## Quick start

```bash
# Download (macOS arm64; other platforms: docs/install.md)
curl -sL https://github.com/Leechael/gemini-web-cli/releases/latest/download/gemini-web-cli-darwin-arm64.tar.gz | tar xz

# Import cookies from your browser
./gemini-web-cli import '_ga=...; __Secure-1PSID=...'

# Or pipe from stdin
pbpaste | ./gemini-web-cli import

# Ask a question
./gemini-web-cli ask "What is the capital of France?"
```

## What you get

Text, image, video, music generation, deep research, and notebooks -- all through Gemini's web interface.

```bash
# Text
gemini-web-cli ask "Explain quantum computing"

# Image generation
gemini-web-cli ask --mode image "A cat astronaut on Mars"

# Video generation
gemini-web-cli ask --mode video "A timelapse of a city at night"

# Continue a conversation
gemini-web-cli reply c_abc123 "Tell me more"

# Deep research
gemini-web-cli research run "Compare Rust and Go for systems programming"
gemini-web-cli report c_abc123 --output report.md

# Notebooks
gemini-web-cli notebook create "My Project"
gemini-web-cli notebook add-source 5a088119-... notes.md report.pdf
gemini-web-cli ask --notebook 5a088119-... "Summarize the sources"
```

Full command reference: [docs/cli.md](docs/cli.md).

## Serve: HTTP API + MCP

`serve` runs a local server with an OpenAI-compatible REST API and an MCP endpoint. Point any OpenAI client or MCP-capable editor at it.

```bash
gemini-web-cli serve --state-dir ~/.local/share/gemini-web-cli/serve
```

OpenAI-compatible chat:

```bash
curl http://127.0.0.1:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"model":"gemini-2.5-flash","messages":[{"role":"user","content":"hello"}]}'
```

MCP (Streamable HTTP at `/mcp`):

```toml
# Codex: ~/.codex/config.toml
[mcp_servers.gemini-web-cli]
url = "http://127.0.0.1:8080/mcp"
```

```json
// Cursor: .cursor/mcp.json
{
  "mcpServers": {
    "gemini-web-cli": {
      "type": "streamable-http",
      "url": "http://127.0.0.1:8080/mcp"
    }
  }
}
```

Interactive docs: `http://127.0.0.1:8080/docs`.

## Docker

```bash
# Import cookies (no host binary needed)
docker run --rm --user "$(id -u):$(id -g)" -v "$PWD:/out" \
  ghcr.io/leechael/gemini-web-cli:latest \
  import '_ga=...; __Secure-1PSID=...' -o /out/cookies.json

# Run the server
docker run --user "$(id -u):$(id -g)" -p 8080:8080 \
  -v "$PWD/cookies.json:/cookies:ro" \
  -v "$PWD/state:/state" \
  ghcr.io/leechael/gemini-web-cli:latest
```

Multi-account: mount a directory of `*.json` files at `/cookies`.

Env, volumes, and logs: [docs/docker.md](docs/docker.md).

## Docs

| | |
|---|---|
| [Install](docs/install.md) | Binary download for all platforms |
| [CLI](docs/cli.md) | Commands, cookies, global flags |
| [Serve](docs/serve.md) | Server flags, multi-account, cookie priority |
| [HTTP API](docs/http-api.md) | REST endpoints, chat state mapping, research |
| [MCP](docs/mcp.md) | Tools, client config for Codex / Cursor / VS Code / Claude Desktop |
| [Docker](docs/docker.md) | Image, volumes, env overrides |

## Acknowledgments

Built on the reverse-engineering in [Gemini-API](https://github.com/HanaokaYuzu/Gemini-API) by [@HanaokaYuzu](https://github.com/HanaokaYuzu).

## License

Same as [Gemini-API](https://github.com/HanaokaYuzu/Gemini-API).
