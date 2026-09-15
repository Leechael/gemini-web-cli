# Binary install

Download from [GitHub Releases](https://github.com/Leechael/gemini-web-cli/releases/latest).

| OS | Arch | Artifact |
|---|---|---|
| macOS | arm64 | `gemini-web-cli-darwin-arm64.tar.gz` |
| macOS | amd64 | `gemini-web-cli-darwin-amd64.tar.gz` |
| Linux | amd64 | `gemini-web-cli-linux-amd64.tar.gz` |
| Linux | arm64 | `gemini-web-cli-linux-arm64.tar.gz` |
| Windows | amd64 | `gemini-web-cli-windows-amd64.zip` |
| Windows | arm64 | `gemini-web-cli-windows-arm64.zip` |

```bash
curl -sL https://github.com/Leechael/gemini-web-cli/releases/latest/download/gemini-web-cli-darwin-arm64.tar.gz | tar xz
./gemini-web-cli import '_ga=...; __Secure-1PSID=...'
./gemini-web-cli serve --state-dir ~/.local/share/gemini-web-cli/serve
```

Windows: download the `.zip`, extract, run `gemini-web-cli.exe`.

Default listen address is `127.0.0.1:8080`. Cookie resolution, flags, and multi-account: [serve.md](serve.md). Commands: [cli.md](cli.md).
