# gemini-web-cli

在终端使用 [Google Gemini](https://gemini.google.com)，或将其作为 API 服务运行 -- 不需要 API key，只需浏览器 cookie。

[English](README.md)

## 快速开始

```bash
# 下载（macOS arm64；其他平台见 docs/install.zh.md）
curl -sL https://github.com/Leechael/gemini-web-cli/releases/latest/download/gemini-web-cli-darwin-arm64.tar.gz | tar xz

# 导入浏览器 cookie
./gemini-web-cli import '_ga=...; __Secure-1PSID=...'

# 也可以通过 stdin 传入
pbpaste | ./gemini-web-cli import

# 提问
./gemini-web-cli ask "法国的首都是哪里？"
```

## 功能一览

文本、图片、视频、音乐生成，深度研究，笔记本 -- 全部通过 Gemini 网页接口实现。

```bash
# 文本
gemini-web-cli ask "解释量子计算"

# 图片生成
gemini-web-cli ask --mode image "火星上的猫宇航员"

# 视频生成
gemini-web-cli ask --mode video "城市夜景延时摄影"

# 继续对话
gemini-web-cli reply c_abc123 "再详细说说"

# 深度研究
gemini-web-cli research run "比较 Rust 和 Go 在系统编程中的优劣"
gemini-web-cli report c_abc123 --output report.md

# 笔记本
gemini-web-cli notebook create "My Project"
gemini-web-cli notebook add-source 5a088119-... notes.md report.pdf
gemini-web-cli ask --notebook 5a088119-... "总结这些资料"
```

完整命令参考：[docs/cli.zh.md](docs/cli.zh.md)。

## Serve：HTTP API + MCP

`serve` 启动一个本地服务，提供 OpenAI 兼容的 REST API 和 MCP 端点。任何 OpenAI 客户端或支持 MCP 的编辑器都可以直接接入。

```bash
gemini-web-cli serve --state-dir ~/.local/share/gemini-web-cli/serve
```

OpenAI 兼容的聊天接口：

```bash
curl http://127.0.0.1:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"model":"gemini-2.5-flash","messages":[{"role":"user","content":"hello"}]}'
```

MCP（Streamable HTTP，路径 `/mcp`）：

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

交互式文档：`http://127.0.0.1:8080/docs`。

## Docker

```bash
# 导入 cookie（不需要在宿主机安装二进制）
docker run --rm --user "$(id -u):$(id -g)" -v "$PWD:/out" \
  ghcr.io/leechael/gemini-web-cli:latest \
  import '_ga=...; __Secure-1PSID=...' -o /out/cookies.json

# 启动服务
docker run --user "$(id -u):$(id -g)" -p 8080:8080 \
  -v "$PWD/cookies.json:/cookies:ro" \
  -v "$PWD/state:/state" \
  ghcr.io/leechael/gemini-web-cli:latest
```

多账号：将多个 `*.json` 文件放在一个目录，挂载到 `/cookies`。

环境变量、卷挂载和日志：[docs/docker.zh.md](docs/docker.zh.md)。

## 文档

| | |
|---|---|
| [安装](docs/install.zh.md) | 各平台二进制下载 |
| [CLI](docs/cli.zh.md) | 命令、cookie 配置、全局参数 |
| [Serve](docs/serve.zh.md) | 服务参数、多账号、cookie 优先级 |
| [HTTP API](docs/http-api.zh.md) | REST 端点、会话状态映射、研究 |
| [MCP](docs/mcp.zh.md) | 工具列表、Codex / Cursor / VS Code / Claude Desktop 配置 |
| [Docker](docs/docker.zh.md) | 镜像、卷挂载、环境变量 |

## 致谢

基于 [@HanaokaYuzu](https://github.com/HanaokaYuzu) 的 [Gemini-API](https://github.com/HanaokaYuzu/Gemini-API) 逆向工程成果构建。

## 许可证

与 [Gemini-API](https://github.com/HanaokaYuzu/Gemini-API) 相同。
