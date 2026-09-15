# Serve

`gemini-web-cli serve` 启动一个进程，包含：

- OpenAI 兼容 REST API `/v1/` -- [http-api.zh.md](http-api.zh.md)
- MCP `/mcp` -- [mcp.zh.md](mcp.zh.md)
- Swagger UI `/docs`

```bash
gemini-web-cli serve [flags]
```

Docker 下默认值不同（监听 `0.0.0.0`，cookie `/cookies`，状态 `/state`）：[docker.zh.md](docker.zh.md)。

## 参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `--port` | 监听端口（或 `GEMINI_WEB_CLI_PORT`） | `8080` |
| `--host` | 绑定地址（或 `GEMINI_WEB_CLI_HOST`） | `127.0.0.1` |
| `--api-key` | `/v1` 端点的 API key（或 `GEMINI_WEB_CLI_API_KEY`） | -- |
| `--expose-thoughts` | API 响应中包含模型推理过程（或 `GEMINI_WEB_CLI_EXPOSE_THOUGHTS=1`） | `false` |
| `--state-dir` | 状态目录（cookie 查找 + 会话映射持久化；或 `GEMINI_WEB_CLI_STATE_DIR`） | -- |
| `--mcp-default-model` | MCP 工具调用未指定 `model` 时的默认模型 | -- |
| `--verbose` | stderr 调试日志（或 `GEMINI_WEB_CLI_VERBOSE=1`） | `false` |
| `--rpc-log` | 启用 RPC 请求/响应日志（或 `GEMINI_WEB_CLI_RPC_LOG=1`；目录通过 `GEMINI_WEB_CLI_RPC_LOG_DIR` 设置） | `false` |

启动时会打印 cookie 来源、监听地址和会话映射路径。

## RPC 日志

`--rpc-log` / `GEMINI_WEB_CLI_RPC_LOG=1` 记录发往 Gemini 的请求和响应。默认关闭。`GEMINI_WEB_CLI_RPC_LOG_DIR` 修改目录（默认 `data/rpc_logs`；Docker 下 `/state/rpc_logs`），但不会开启日志。

布局：每日 NDJSON 索引 `<log-dir>/YYYY-MM-DD.ndjson`，请求/响应体在 `<log-dir>/blobs/YYYY-MM-DD/` 下。Cookie、Authorization、Set-Cookie、batchexecute `at`、初始化 access-token 和 session-ID 值已脱敏。提示词、上传内容和模型响应仍可读。超过 7 天的文件在启动和日志轮转时自动清理。

## Cookie 优先级

1. `--cookies-json` 参数
2. `<state-dir>/cookies.json`
3. `$GEMINI_WEB_COOKIES_JSON_PATH`
4. 自动发现的 `cookies.json` 路径
5. `GEMINI_SECURE_1PSID` / `GEMINI_SECURE_1PSIDTS`

已有 cookie 不会被复制到 `--state-dir`。

## 多账号

`serve` 支持多个 Google 账号以分散频率限制。新对话、研究任务和笔记本按轮询分配；失败时（如 HTTP 429）自动切换到下一个账号。后续操作（会话/研究/笔记本 ID 的追问）保持在创建时的账号上（创建时记录，重启后重新探测）。流式响应仅在第一个 delta 之前可以故障转移。

```bash
gemini-web-cli import '<cookies_alice>' -o accounts/alice.json
gemini-web-cli import '<cookies_bob>' -o accounts/bob.json

gemini-web-cli serve --cookies-json accounts/
# 等同于：--cookies-json accounts/alice.json --cookies-json accounts/bob.json
```

`$GEMINI_WEB_COOKIES_JSON_PATH` 接受路径列表（Linux/macOS 用 `:`）或目录。

单次命令（`ask`、`reply` 等）和 `status --cookies-only` 使用找到的第一个 cookie 文件。`gemini_research_list` 会合并所有账号的报告（按时间倒序）；不支持跨账号分页游标。
