# CLI

## Cookies

Required cookie: `__Secure-1PSID`. `__Secure-1PSIDTS` is recommended.

Resolution order:

1. `--cookies-json` (file, directory of `*.json`, or repeatable)
2. `$GEMINI_WEB_COOKIES_JSON_PATH` (OS path-list or directory)
3. `./cookies.json`
4. `~/.config/gemini-web-cli/cookies.json`
5. `/etc/gemini-web-cli/cookies.json`

Accepted JSON shapes:

- `{"cookies": {"name": "value", ...}}` (`import` output)
- `{"name": "value", ...}`
- `[{"name": "...", "value": "...", ...}, ...]`
- `{"cookies": [{"name": "...", "value": "...", ...}, ...]}`

One-shot commands use the first resolved file. `serve` can load every file: [serve.md](serve.md).

## import

```bash
gemini-web-cli import '_ga=GA1.1.123; __Secure-1PSID=g.a000...; SID=abc...'
gemini-web-cli import '_ga=...' -o path/to/cookies.json
pbpaste | gemini-web-cli import
gemini-web-cli import - -o cookies.json
echo '__Secure-1PSID=...' | gemini-web-cli import
```

TTY without arguments prints an error instead of hanging for keyboard input.

## ask / reply

```bash
gemini-web-cli ask "Explain quantum computing"
gemini-web-cli ask --no-stream "What is 2+2?"
gemini-web-cli ask --mode image "Draw a sunset"
gemini-web-cli ask --mode image -f photo.jpg "Make this photorealistic"
gemini-web-cli ask --mode video "A cat walking in slow motion"
gemini-web-cli ask --mode music "A short jazz melody"
gemini-web-cli ask --mode image-to-video -f photo.jpg "Animate this photo"
gemini-web-cli ask -f image.png "What's in this image?"
gemini-web-cli ask -f a.pdf -f b.pdf "Compare these documents"

gemini-web-cli reply c_abc123 "Tell me more"
gemini-web-cli reply --mode video c_abc123 "Now generate a video of that scene"
```

`--mode`: `auto` (default), `text`, `image`, `video`, `image-to-video`, `music`. Output includes text, any media URLs, and the chat ID.

## list / get / download / progress

```bash
gemini-web-cli list
gemini-web-cli list --cursor <cursor>

gemini-web-cli get c_abc123
gemini-web-cli get c_abc123 --max-turns 10 --output chat.txt
gemini-web-cli get c_abc123 --cursor <cursor>

gemini-web-cli download "https://lh3.googleusercontent.com/..." -o image.png
gemini-web-cli download c_abc123 -o output.png          # output_1.png, output_2.mp4, ...
gemini-web-cli download c_abc123 2 -o video.mp4
gemini-web-cli download --poll "https://contribution.usercontent.google.com/download?..."

gemini-web-cli progress c_abc123
```

`get` prints `user` / `agent` turns with request IDs. Media numbers match `download`'s index. Video/music downloads poll HTTP 206 when fetching by chat ID.

## notebook

```bash
gemini-web-cli notebook create "My Project"
gemini-web-cli notebook get 5a088119-...
gemini-web-cli notebook chats 5a088119-...
gemini-web-cli notebook add-source 5a088119-... notes.md report.pdf
gemini-web-cli notebook add-url 5a088119-... https://example.com/article
gemini-web-cli notebook remove-source notebooks/5a088119-.../sources/fa1beeca-...

gemini-web-cli ask --notebook 5a088119-... "Summarize the sources in one sentence"
gemini-web-cli reply --notebook 5a088119-... c_abc123 "Tell me more"
```

## research / report

```bash
gemini-web-cli research run "Compare Rust and Go for systems programming"
gemini-web-cli research list
gemini-web-cli research list --count 20 --cursor <cursor> --json
gemini-web-cli report c_abc123
gemini-web-cli report c_abc123 --output report.md
```

## chat / expand-prompt / models / status

```bash
gemini-web-cli chat meta c_abc123
gemini-web-cli chat turn c_abc123 <requestId> --json
gemini-web-cli expand-prompt "A sunset over the ocean"
gemini-web-cli models
gemini-web-cli status
gemini-web-cli status --cookies-only --cookies-json cookies.json
```

`models` lists names for `--model`. `unspecified` lets Gemini auto-select. Dynamic models come from the current account when cookies are available.

## debug

```bash
gemini-web-cli debug rpc otAQ7b
gemini-web-cli debug rpc MaZiqc --payload '[13,null,[1,null,1]]'
gemini-web-cli debug rpc hNvQHb --payload '["c_abc",10]' --source-cid c_abc
gemini-web-cli debug rpc cYRIkd --payload '["en"]' --pretty
gemini-web-cli debug housekeeping heartbeat
gemini-web-cli debug housekeeping list-gems --pretty
```

Housekeeping names: `heartbeat`, `ui-heartbeat`, `set-lang`, `ma-gu-ac`, `list-gems`, `bulk-log`, `log-event`, `log-model-select`.

## Global flags

| Flag | Description | Default |
|------|-------------|---------|
| `--cookies-json` | Cookie JSON file or directory (repeatable) | `$GEMINI_WEB_COOKIES_JSON_PATH` |
| `--model` | Model name (`models`) | `unspecified` |
| `--proxy` | HTTP/SOCKS proxy URL | `$HTTPS_PROXY` |
| `--account-index` | Google account index (`/u/2`) | — |
| `--verbose` | Debug logging to stderr | `$GEMINI_WEB_CLI_VERBOSE=1` |
| `--rpc-log` | RPC bodies to disk (cookies redacted) | `$GEMINI_WEB_CLI_RPC_LOG=1` |
| `--no-persist` | Do not write cookies back | `false` |
| `--request-timeout` | HTTP timeout in seconds | `300` |

## E2E

```bash
./scripts/e2e-test.sh cookies.json
```
