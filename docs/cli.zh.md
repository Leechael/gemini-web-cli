# CLI

## Cookie

必需 cookie：`__Secure-1PSID`。建议同时提供 `__Secure-1PSIDTS`。

查找顺序：

1. `--cookies-json`（文件、`*.json` 目录，可重复）
2. `$GEMINI_WEB_COOKIES_JSON_PATH`（路径列表或目录）
3. `./cookies.json`
4. `~/.config/gemini-web-cli/cookies.json`
5. `/etc/gemini-web-cli/cookies.json`

支持的 JSON 格式：

- `{"cookies": {"name": "value", ...}}`（`import` 输出格式）
- `{"name": "value", ...}`
- `[{"name": "...", "value": "...", ...}, ...]`
- `{"cookies": [{"name": "...", "value": "...", ...}, ...]}`

单次命令使用找到的第一个文件。`serve` 可以加载所有文件：[serve.zh.md](serve.zh.md)。

## import

```bash
gemini-web-cli import '_ga=GA1.1.123; __Secure-1PSID=g.a000...; SID=abc...'
gemini-web-cli import '_ga=...' -o path/to/cookies.json
pbpaste | gemini-web-cli import
gemini-web-cli import - -o cookies.json
echo '__Secure-1PSID=...' | gemini-web-cli import
```

TTY 下不传参数会报错，不会挂住等键盘输入。

## ask / reply

```bash
gemini-web-cli ask "解释量子计算"
gemini-web-cli ask --no-stream "1+1 等于几？"
gemini-web-cli ask --mode image "画一幅日落"
gemini-web-cli ask --mode image -f photo.jpg "把这张图变成写实风格"
gemini-web-cli ask --mode video "一只猫慢动作走路"
gemini-web-cli ask --mode music "一段短爵士旋律"
gemini-web-cli ask --mode image-to-video -f photo.jpg "让这张照片动起来"
gemini-web-cli ask -f image.png "这张图里有什么？"
gemini-web-cli ask -f a.pdf -f b.pdf "比较这两个文档"

gemini-web-cli reply c_abc123 "再详细说说"
gemini-web-cli reply --mode video c_abc123 "把这个场景生成视频"
```

`--mode`：`auto`（默认）、`text`、`image`、`video`、`image-to-video`、`music`。输出包含文本、媒体 URL 和会话 ID。

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

`get` 按 `user` / `agent` 输出对话轮次和 request ID。媒体编号与 `download` 的索引对应。通过会话 ID 下载视频/音乐时会自动轮询 HTTP 206。

## notebook

```bash
gemini-web-cli notebook create "My Project"
gemini-web-cli notebook get 5a088119-...
gemini-web-cli notebook chats 5a088119-...
gemini-web-cli notebook add-source 5a088119-... notes.md report.pdf
gemini-web-cli notebook add-url 5a088119-... https://example.com/article
gemini-web-cli notebook remove-source notebooks/5a088119-.../sources/fa1beeca-...

gemini-web-cli ask --notebook 5a088119-... "用一句话总结这些资料"
gemini-web-cli reply --notebook 5a088119-... c_abc123 "再详细说说"
```

## research / report

```bash
gemini-web-cli research run "比较 Rust 和 Go 在系统编程中的优劣"
gemini-web-cli research list
gemini-web-cli research list --count 20 --cursor <cursor> --json
gemini-web-cli report c_abc123
gemini-web-cli report c_abc123 --output report.md
```

## chat / expand-prompt / models / status

```bash
gemini-web-cli chat meta c_abc123
gemini-web-cli chat turn c_abc123 <requestId> --json
gemini-web-cli expand-prompt "海上日落"
gemini-web-cli models
gemini-web-cli status
gemini-web-cli status --cookies-only --cookies-json cookies.json
```

`models` 列出可用于 `--model` 的模型名称。`unspecified` 让 Gemini 自动选择。动态模型列表来自当前账号（需要可用 cookie）。

## debug

```bash
gemini-web-cli debug rpc otAQ7b
gemini-web-cli debug rpc MaZiqc --payload '[13,null,[1,null,1]]'
gemini-web-cli debug rpc hNvQHb --payload '["c_abc",10]' --source-cid c_abc
gemini-web-cli debug rpc cYRIkd --payload '["en"]' --pretty
gemini-web-cli debug housekeeping heartbeat
gemini-web-cli debug housekeeping list-gems --pretty
```

Housekeeping 名称：`heartbeat`、`ui-heartbeat`、`set-lang`、`ma-gu-ac`、`list-gems`、`bulk-log`、`log-event`、`log-model-select`。

## 全局参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `--cookies-json` | Cookie JSON 文件或目录（可重复） | `$GEMINI_WEB_COOKIES_JSON_PATH` |
| `--model` | 模型名称（`models`） | `unspecified` |
| `--proxy` | HTTP/SOCKS 代理 URL | `$HTTPS_PROXY` |
| `--account-index` | Google 账号索引（`/u/2`） | -- |
| `--verbose` | 输出调试日志到 stderr | `$GEMINI_WEB_CLI_VERBOSE=1` |
| `--rpc-log` | RPC 请求/响应写入磁盘（cookie 已脱敏） | `$GEMINI_WEB_CLI_RPC_LOG=1` |
| `--no-persist` | 不回写更新后的 cookie | `false` |
| `--request-timeout` | HTTP 超时（秒） | `300` |

## E2E

```bash
./scripts/e2e-test.sh cookies.json
```
