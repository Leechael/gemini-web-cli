# Pi provider for Gemini Web

Use an internal `gemini-web-cli serve` instance as a Pi provider. The extension discovers available models, preserves Gemini conversations with `chat_id`, and exposes Deep Research as Pi tools.

## 1. Start the server

```bash
gemini-web-cli serve --host 127.0.0.1 --port 8080
```

The extension targets an internal server without authentication.

## 2. Install the extension

From this repository:

```bash
pi install /path/to/gemini-web-cli/pi-provider-gemini-web
```

To load it temporarily during development:

```bash
pi -e /path/to/gemini-web-cli/pi-provider-gemini-web
```

## 3. Configure the server address

### Project configuration

Create `<project>/.pi/pi-provider-gemini-web.json`:

```json
{
  "baseUrl": "http://127.0.0.1:8080"
}
```

Pi reads this file only after the project is trusted.

### Global configuration

Create `~/.pi/agent/extensions/pi-provider-gemini-web.json` with the same content:

```json
{
  "baseUrl": "http://127.0.0.1:8080"
}
```

When `PI_CODING_AGENT_DIR` is set, the global path is:

```text
$PI_CODING_AGENT_DIR/extensions/pi-provider-gemini-web.json
```

Project configuration replaces global configuration for that project. `baseUrl` must be an absolute `http` or `https` URL and must not include `/v1`.

## 4. Select a model

Start Pi and select a discovered model:

```text
/model gemini-web/gemini-3.5-flash
```

For a project-only configuration, the built-in startup catalog includes:

- `gemini-3.5-flash`
- `gemini-3.1-pro`
- `gemini-3.1-flash-lite`

After the session starts, the extension refreshes this catalog from:

```text
GET <baseUrl>/v1/models
```

## Conversation continuity

The first response records the Gemini chat ID in the Pi assistant message. Later turns send that ID as `chat_id`, so resumed Pi sessions continue the same Gemini conversation. The first request after `/fork` or `/clone` starts a new Gemini conversation instead of modifying the parent branch.

## Deep Research tools

The extension registers:

- `gemini_research_create` — submit a research task
- `gemini_research_status` — poll its state
- `gemini_research_result` — fetch the completed report and sources

Typical flow:

```text
gemini_research_create
        ↓
gemini_research_status until state = done
        ↓
gemini_research_result
```

All tools use the same effective `baseUrl` as the provider.

## Development

```bash
cd pi-provider-gemini-web
npm install
npm test
npm run check
npm run lint
npm run format:check
```
