# 二进制安装

从 [GitHub Releases](https://github.com/Leechael/gemini-web-cli/releases/latest) 下载。

| 系统 | 架构 | 文件 |
|------|------|------|
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

Windows：下载 `.zip`，解压后运行 `gemini-web-cli.exe`。

默认监听地址为 `127.0.0.1:8080`。Cookie 配置、参数和多账号：[serve.zh.md](serve.zh.md)。命令参考：[cli.zh.md](cli.zh.md)。
