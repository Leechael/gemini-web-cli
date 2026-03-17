#!/bin/bash
###############
# 脚本名称: refresh-cookies.sh
# 脚本作用: 通过 Chrome CDP 刷新 Gemini cookies
# 使用方法: ./refresh-cookies.sh
# 功能说明: 1. 连接本地 Chrome 调试端口 2. 提取 Gemini cookie 3. 导入到 gemini-web-cli
# 注意事项: - 需要 Chrome 开启调试端口 9222 且有 gemini.google.com 标签页
###############

set -e

CLI="$HOME/Sites/gemini-web-cli/bin/gemini-web-cli"
COOKIES_FILE="$HOME/.config/gemini-web-cli/cookies.json"
CDP_PORT=9222

# 检查 Chrome 调试端口
if ! curl -s "http://localhost:${CDP_PORT}/json/list" > /dev/null 2>&1; then
    echo "Error: Chrome 调试端口未开启"
    echo "请先运行: ~/Sites/tools/start-chrome-debug.sh 并在调试 Chrome 中登录 Gemini"
    exit 1
fi

# 通过 CDP 获取 Gemini cookie
COOKIES=$(python3 << 'PYEOF'
import http.client
import json
import struct
import base64
import socket
import sys

CDP_PORT = 9222

# 1. 找到 Gemini 标签页
conn = http.client.HTTPConnection("localhost", CDP_PORT)
conn.request("GET", "/json/list")
pages = json.loads(conn.getresponse().read())
conn.close()

ws_url = None
for p in pages:
    if p.get("type") == "page" and "gemini.google.com" in p.get("url", ""):
        ws_url = p.get("webSocketDebuggerUrl")
        break

if not ws_url:
    print("ERROR: No Gemini tab found", file=sys.stderr)
    sys.exit(1)

# 2. WebSocket handshake (RFC 6455, no external deps)
host_port = ws_url.split("//")[1].split("/")[0]
path = "/" + "/".join(ws_url.split("//")[1].split("/")[1:])
host, port = host_port.split(":") if ":" in host_port else (host_port, "80")

sock = socket.create_connection((host, int(port)), timeout=10)
key = base64.b64encode(b"gemini-refresh-key!").decode()
handshake = (
    f"GET {path} HTTP/1.1\r\n"
    f"Host: {host}:{port}\r\n"
    f"Upgrade: websocket\r\n"
    f"Connection: Upgrade\r\n"
    f"Sec-WebSocket-Key: {key}\r\n"
    f"Sec-WebSocket-Version: 13\r\n"
    f"\r\n"
)
sock.sendall(handshake.encode())

# Read handshake response
resp = b""
while b"\r\n\r\n" not in resp:
    resp += sock.recv(4096)

# 3. Send getAllCookies via WebSocket
def ws_send(sock, data):
    payload = data.encode("utf-8")
    frame = bytearray([0x81])  # text frame, FIN
    length = len(payload)
    mask_key = b"\x12\x34\x56\x78"
    if length < 126:
        frame.append(0x80 | length)
    elif length < 65536:
        frame.append(0x80 | 126)
        frame.extend(struct.pack(">H", length))
    else:
        frame.append(0x80 | 127)
        frame.extend(struct.pack(">Q", length))
    frame.extend(mask_key)
    masked = bytearray(b ^ mask_key[i % 4] for i, b in enumerate(payload))
    frame.extend(masked)
    sock.sendall(frame)

def ws_recv(sock):
    header = sock.recv(2)
    if len(header) < 2:
        return ""
    length = header[1] & 0x7F
    if length == 126:
        length = struct.unpack(">H", sock.recv(2))[0]
    elif length == 127:
        length = struct.unpack(">Q", sock.recv(8))[0]
    data = b""
    while len(data) < length:
        chunk = sock.recv(length - len(data))
        if not chunk:
            break
        data += chunk
    return data.decode("utf-8", errors="replace")

msg = json.dumps({"id": 1, "method": "Network.getAllCookies"})
ws_send(sock, msg)
raw = ws_recv(sock)
sock.close()

result = json.loads(raw)
target_names = {"__Secure-1PSID", "__Secure-1PSIDTS"}
parts = []
for c in result.get("result", {}).get("cookies", []):
    if c["name"] in target_names:
        parts.append(f"{c['name']}={c['value']}")

if parts:
    print("; ".join(parts))
else:
    print("ERROR: Target cookies not found", file=sys.stderr)
    sys.exit(1)
PYEOF
)

if [ -z "$COOKIES" ] || [[ "$COOKIES" == ERROR* ]]; then
    echo "Error: 无法获取 Cookie"
    exit 1
fi

# 确保目录存在
mkdir -p "$(dirname "$COOKIES_FILE")"

# 导入 Cookie
"$CLI" import "$COOKIES" --cookies-json "$COOKIES_FILE"

# 验证
if "$CLI" --cookies-json "$COOKIES_FILE" inspect > /dev/null 2>&1; then
    echo "Cookie 刷新成功"
else
    echo "Cookie 验证失败"
    exit 1
fi
