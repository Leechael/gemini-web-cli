# HTTP API

`gemini-web-cli serve` 提供 OpenAI 兼容的 REST API，以及研究和笔记本功能。交互式文档：`http://127.0.0.1:8080/docs`（OpenAPI 规范：`/openapi.json`）。

## 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/v1/models` | 列出可用模型 |
| `POST` | `/v1/chat/completions` | OpenAI 兼容的聊天补全 |
| `POST` | `/v1/research` | 提交深度研究任务 |
| `GET` | `/v1/research/{id}` | 研究摘要 |
| `GET` | `/v1/research/{id}/status` | 轮询研究状态 |
| `GET` | `/v1/research/{id}/result` | 获取完成的报告 |
| `POST` | `/v1/notebooks` | 创建笔记本 |
| `GET` | `/v1/notebooks/{id}` | 获取笔记本 |
| `GET` | `/v1/notebooks/{id}/chats` | 列出笔记本中的对话 |
| `POST` | `/v1/notebooks/{id}/sources` | 添加文件或 URL 来源 |
| `DELETE` | `/v1/notebooks/{id}/sources/{sid}` | 移除来源 |
| `GET` | `/docs` | Swagger UI |
| `GET` | `/openapi.json` | OpenAPI 规范 |

设置 `--api-key` 或 `GEMINI_WEB_CLI_API_KEY` 后，`/v1/` 需要 `Authorization: Bearer <key>` 或 `X-API-Key: <key>`。`/mcp` 不受此限制。

## 聊天补全

`POST /v1/chat/completions` 兼容 OpenAI 格式。标准客户端无需修改即可使用。扩展字段：响应中的 `chat_id` 是 Gemini 会话 ID。

不支持图片部分、tool calls 和 function calls。

```bash
curl http://127.0.0.1:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"model":"gemini-3.8-flash","messages":[{"role":"user","content":"hello"}]}'
```

流式输出：设置 `"stream": true`。继续对话：发送上次响应中的 `chat_id`。

## 会话状态映射

设置 `--state-dir` / `GEMINI_WEB_CLI_STATE_DIR` 后，服务会持久化 `<state-dir>/chat-map.pb`。它对 OpenAI `messages` 历史做哈希匹配已有的 Gemini 对话并自动续接。否则会用拼合的文本提示开始新对话。

条目分为 **verified**（由本服务创建）和 **synthetic**（从客户端历史推断）。不支持分叉对话。

## 研究

```bash
curl -X POST http://127.0.0.1:8080/v1/research \
  -H 'Content-Type: application/json' \
  -d '{"prompt":"在这里输入研究主题"}'

curl http://127.0.0.1:8080/v1/research/{id}
curl http://127.0.0.1:8080/v1/research/{id}/status
curl http://127.0.0.1:8080/v1/research/{id}/result
```

## 相关文档

- [Serve 参数](serve.zh.md)
- [MCP](mcp.zh.md)
