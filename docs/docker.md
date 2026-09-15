# Docker

Release tags publish a multi-arch image (`linux/amd64`, `linux/arm64`) to GHCR: `ghcr.io/leechael/gemini-web-cli`.

The image entrypoint is the `gemini-web-cli` binary. Default command: `serve`.

## Cookies

Write a cookie file with the same image (no host CLI):

```bash
docker run --rm --user "$(id -u):$(id -g)" -v "$PWD:/out" \
  ghcr.io/leechael/gemini-web-cli:latest \
  import '_ga=...; __Secure-1PSID=...' -o /out/cookies.json
```

`--user` is required so the file lands on the host owned by you (`import` writes `0600`). Repeat into `accounts/alice.json`, `accounts/bob.json`, … for multi-account.

## Run

`--user` matches the host uid so `0600` cookie files and the `/state` mount are readable/writable.

Single account:

```bash
docker run --user "$(id -u):$(id -g)" -p 8080:8080 \
  -v "$PWD/cookies.json:/cookies:ro" \
  -v "$PWD/state:/state" \
  ghcr.io/leechael/gemini-web-cli:latest
```

Multiple accounts (directory of `*.json`, name order):

```bash
docker run --user "$(id -u):$(id -g)" -p 8080:8080 \
  -v "$PWD/accounts:/cookies:ro" \
  -v "$PWD/state:/state" \
  ghcr.io/leechael/gemini-web-cli:latest
```

`-p` publishes the container port. The process listens on `0.0.0.0:8080` inside the container.

One-off commands:

```bash
docker run --rm -v "$PWD/cookies.json:/cookies:ro" \
  ghcr.io/leechael/gemini-web-cli:latest --help
```

## Defaults

| | Image default |
|---|---|
| Command | `serve` |
| Listen | `0.0.0.0:8080` |
| Cookies | `/cookies` (file or directory of `*.json`) |
| State | `/state` (`chat-map.pb`; ephemeral unless you mount it) |
| API key | unset |
| RPC log dir | `/state/rpc_logs` (still off until enabled) |

## Environment

| Variable | Purpose | Image default |
|---|---|---|
| `GEMINI_WEB_CLI_HOST` | Bind address | `0.0.0.0` |
| `GEMINI_WEB_CLI_PORT` | Bind port | `8080` |
| `GEMINI_WEB_COOKIES_JSON_PATH` | Cookie file or directory | `/cookies` |
| `GEMINI_WEB_CLI_STATE_DIR` | Chat-map directory | `/state` |
| `GEMINI_WEB_CLI_API_KEY` | Auth for `/v1` | unset |
| `GEMINI_WEB_CLI_VERBOSE` | Debug logs on stderr | unset |
| `GEMINI_WEB_CLI_RPC_LOG` | Write RPC bodies to disk | unset |
| `GEMINI_WEB_CLI_RPC_LOG_DIR` | RPC log directory | `/state/rpc_logs` |
| `GEMINI_WEB_CLI_EXPOSE_THOUGHTS` | Include reasoning in API responses | unset |

Flags still win over env. Local CLI host stays `127.0.0.1` when these image env values are absent.

Example:

```bash
docker run -p 9000:9000 \
  -e GEMINI_WEB_CLI_PORT=9000 \
  -e GEMINI_WEB_CLI_API_KEY=your-secret \
  -v "$PWD/accounts:/cookies:ro" \
  -v "$PWD/state:/state" \
  ghcr.io/leechael/gemini-web-cli:latest
```

Published ports are reachable by anyone who can reach the host. Set `GEMINI_WEB_CLI_API_KEY` when binding beyond localhost. `/mcp` is never API-key protected.

## Logs

`docker logs` prints redacted operational lines: account ready/failover, HTTP access (`method path status dur`) without query strings, headers, or bodies.

```bash
docker logs <container>
docker run -e GEMINI_WEB_CLI_VERBOSE=1 ...
docker run -e GEMINI_WEB_CLI_RPC_LOG=1 ...   # /state/rpc_logs; cookies redacted, prompts still present
```

## Next

- [MCP](mcp.md)
- [HTTP API](http-api.md)
- [Serve flags and multi-account](serve.md)
