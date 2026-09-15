# Docker

Release tag 会发布多架构镜像（`linux/amd64`、`linux/arm64`）到 GHCR：`ghcr.io/leechael/gemini-web-cli`。

镜像入口是 `gemini-web-cli` 二进制。默认命令：`serve`。

## Cookie

用同一个镜像生成 cookie 文件（不需要在宿主机安装）：

```bash
docker run --rm --user "$(id -u):$(id -g)" -v "$PWD:/out" \
  ghcr.io/leechael/gemini-web-cli:latest \
  import '_ga=...; __Secure-1PSID=...' -o /out/cookies.json
```

`--user` 确保文件以你的用户身份写入（`import` 写入权限为 `0600`）。多账号：分别写入 `accounts/alice.json`、`accounts/bob.json` 等。

## 运行

`--user` 匹配宿主机 uid，确保 `0600` 权限的 cookie 文件和 `/state` 挂载可读写。

单账号：

```bash
docker run --user "$(id -u):$(id -g)" -p 8080:8080 \
  -v "$PWD/cookies.json:/cookies:ro" \
  -v "$PWD/state:/state" \
  ghcr.io/leechael/gemini-web-cli:latest
```

多账号（`*.json` 文件目录，按文件名排序）：

```bash
docker run --user "$(id -u):$(id -g)" -p 8080:8080 \
  -v "$PWD/accounts:/cookies:ro" \
  -v "$PWD/state:/state" \
  ghcr.io/leechael/gemini-web-cli:latest
```

`-p` 发布容器端口。容器内进程监听 `0.0.0.0:8080`。

一次性命令：

```bash
docker run --rm -v "$PWD/cookies.json:/cookies:ro" \
  ghcr.io/leechael/gemini-web-cli:latest --help
```

## 默认值

| | 镜像默认 |
|---|---|
| 命令 | `serve` |
| 监听 | `0.0.0.0:8080` |
| Cookie | `/cookies`（文件或 `*.json` 目录） |
| 状态 | `/state`（`chat-map.pb`；不挂载则为临时存储） |
| API key | 未设置 |
| RPC 日志目录 | `/state/rpc_logs`（默认关闭，需开启） |

## 环境变量

| 变量 | 用途 | 镜像默认 |
|---|---|---|
| `GEMINI_WEB_CLI_HOST` | 绑定地址 | `0.0.0.0` |
| `GEMINI_WEB_CLI_PORT` | 绑定端口 | `8080` |
| `GEMINI_WEB_COOKIES_JSON_PATH` | Cookie 文件或目录 | `/cookies` |
| `GEMINI_WEB_CLI_STATE_DIR` | 会话映射目录 | `/state` |
| `GEMINI_WEB_CLI_API_KEY` | `/v1` 认证 | 未设置 |
| `GEMINI_WEB_CLI_VERBOSE` | stderr 调试日志 | 未设置 |
| `GEMINI_WEB_CLI_RPC_LOG` | RPC 请求/响应写入磁盘 | 未设置 |
| `GEMINI_WEB_CLI_RPC_LOG_DIR` | RPC 日志目录 | `/state/rpc_logs` |
| `GEMINI_WEB_CLI_EXPOSE_THOUGHTS` | API 响应中包含推理过程 | 未设置 |

命令行参数优先于环境变量。本地 CLI 在这些镜像环境变量不存在时仍使用 `127.0.0.1`。

示例：

```bash
docker run -p 9000:9000 \
  -e GEMINI_WEB_CLI_PORT=9000 \
  -e GEMINI_WEB_CLI_API_KEY=your-secret \
  -v "$PWD/accounts:/cookies:ro" \
  -v "$PWD/state:/state" \
  ghcr.io/leechael/gemini-web-cli:latest
```

发布的端口对能访问宿主机的任何人可见。绑定到 localhost 以外时建议设置 `GEMINI_WEB_CLI_API_KEY`。`/mcp` 不受 API key 保护。

## 日志

`docker logs` 输出脱敏的运行日志：账号就绪/故障转移、HTTP 访问（`method path status dur`），不含查询字符串、请求头或请求体。

```bash
docker logs <container>
docker run -e GEMINI_WEB_CLI_VERBOSE=1 ...
docker run -e GEMINI_WEB_CLI_RPC_LOG=1 ...   # /state/rpc_logs；cookie 已脱敏，提示词仍然可见
```

## 相关文档

- [MCP](mcp.zh.md)
- [HTTP API](http-api.zh.md)
- [Serve 参数与多账号](serve.zh.md)
