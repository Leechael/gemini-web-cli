# MCP

`gemini-web-cli serve` 在 `/mcp` 路径提供 Model Context Protocol 端点（Streamable HTTP，无状态）。

`/mcp` **不受** `--api-key` 保护。请保持监听在 `127.0.0.1`，或在前面加认证代理。

`--mcp-default-model` 设置未指定 `model` 的工具调用的默认模型。如果未设置且调用也未指定，Gemini 自动选择。

## 工具

| 工具 | 说明 |
|------|------|
| `gemini_ask` | 单轮提问；返回 `text` 及生成的图片/视频/媒体 URL。参数：`prompt`（必需）、`model`（可选）、`notebook`（可选，限定到某个笔记本）。 |
| `gemini_research_create` | 提交深度研究任务；返回 `id`、`title`、`eta_text`、`steps`。参数：`prompt`（必需）、`model`（可选）。 |
| `gemini_research_status` | 轮询任务状态（`done`、`running`、`pending_confirm`、`not_research`、`empty`）。参数：`id`（必需）。 |
| `gemini_research_result` | 获取完成的报告文本和来源引用。参数：`id`（必需）。 |
| `gemini_research_list` | 列出已完成的深度研究报告。参数：`count`（可选，默认 `13`）、`cursor`（可选）。 |
| `gemini_research_reply` | 对已有研究对话进行追问；之后轮询 `gemini_research_status`。参数：`id`（必需）、`prompt`（必需）、`model`（可选）。 |
| `gemini_list_models` | 列出可用模型名称和显示名称。无参数。 |
| `gemini_notebook_create` | 创建笔记本；返回 `resource`（`notebooks/<uuid>`）和 `title`。参数：`title`（必需）。 |
| `gemini_notebook_get` | 笔记本标题、图标和来源列表。参数：`id`（必需）。 |
| `gemini_notebook_list_chats` | 列出笔记本中的对话，按时间倒序。参数：`id`（必需）。 |
| `gemini_notebook_add_file_source` | 上传 serve 主机上的本地文件并附加为来源。参数：`id`（必需）、`path`（必需）。 |
| `gemini_notebook_add_url_source` | 附加网页 URL 为来源。参数：`id`（必需）、`url`（必需）。 |
| `gemini_notebook_remove_source` | 移除来源。参数：`source`（必需，`notebooks/<uuid>/sources/<sid>`）。 |

深度研究流程：`gemini_research_create` -> 轮询 `gemini_research_status` 直到 `done` -> `gemini_research_result`。追问用 `gemini_research_reply`。

## 客户端配置

保持 `serve` 运行，客户端即可使用工具。编辑配置后需重启客户端。

### Codex

`~/.codex/config.toml`（或项目 `.codex/config.toml`）：

```toml
[mcp_servers.gemini-web-cli]
url = "http://127.0.0.1:8080/mcp"
```

运行 `codex mcp list` 验证。在会话中 `/mcp` 显示已连接的服务。

### Cursor

`.cursor/mcp.json`：

```json
{
  "mcpServers": {
    "gemini-web-cli": {
      "type": "streamable-http",
      "url": "http://127.0.0.1:8080/mcp"
    }
  }
}
```

### VS Code

`.vscode/mcp.json`：

```json
{
  "servers": {
    "gemini-web-cli": {
      "type": "http",
      "url": "http://127.0.0.1:8080/mcp"
    }
  }
}
```

### Claude Desktop

`claude_desktop_config.json` 只支持 stdio 服务。用 [`mcp-remote`](https://www.npmjs.com/package/mcp-remote) 桥接：

```json
{
  "mcpServers": {
    "gemini-web-cli": {
      "command": "npx",
      "args": ["-y", "mcp-remote", "http://127.0.0.1:8080/mcp"]
    }
  }
}
```

Docker：如果发布了 `8080:8080` 端口，从宿主机访问的 URL 仍然是 `http://127.0.0.1:8080/mcp`。
