# Cookie Auto-Decrypt: 备忘（决定不做）

## 概述

pi-web-access 项目实现了从本地 Chromium 浏览器自动解密 cookie 的功能，用户无需手动导出 cookie。

## 实现细节（来自 pi-web-access）

### 支持的浏览器

**macOS:**
| 浏览器 | 数据目录 | Keychain Service | Keychain Account |
|--------|---------|-----------------|-----------------|
| Chrome | `~/Library/Application Support/Google/Chrome` | `Chrome Safe Storage` | `Chrome` |
| Arc | `~/Library/Application Support/Arc/User Data` | `Arc Safe Storage` | `Arc` |
| Helium | `~/Library/Application Support/net.imput.helium` | `Helium Storage Key` | `Helium` |

**Linux:**
| 浏览器 | 数据目录 | Secret Tool App |
|--------|---------|----------------|
| Chrome | `~/.config/google-chrome` | `chrome` |
| Chromium | `~/.config/chromium` | `chromium` |

### 解密流程

1. **获取加密密码**
   - macOS: `security find-generic-password -w -a <account> -s <service>`
   - Linux: `secret-tool lookup application <app>`（fallback `"peanuts"`）

2. **派生 AES 密钥**: `PBKDF2(password, "saltysalt", iterations, 16, "sha1")`
   - macOS: 1003 iterations
   - Linux: 1 iteration

3. **复制 SQLite 数据库**（避免锁冲突，含 `-wal` 和 `-shm`）到临时目录

4. **查询 cookie**: 按 `host_key` 匹配 Google 域名

5. **解密 cookie 值**:
   - 检查 `v10`/`v11` 前缀（3 字节版本标识）
   - AES-128-CBC，IV = 16 个 `0x20`（空格）字节
   - PKCS7 padding 移除
   - Chrome `meta.version >= 24`: 剥离解密后前 32 字节（hash 前缀）
   - 剥离前导控制字符

### 采集的 Cookie（18 个）

`__Secure-1PSID`, `__Secure-1PSIDTS`, `__Secure-1PSIDCC`, `__Secure-1PAPISID`,
`NID`, `AEC`, `SOCS`, `__Secure-BUCKET`, `__Secure-ENID`, `SID`, `HSID`, `SSID`,
`APISID`, `SAPISID`, `__Secure-3PSID`, `__Secure-3PSIDTS`, `__Secure-3PAPISID`, `SIDCC`

### 扫描的域

`https://gemini.google.com`, `https://accounts.google.com`, `https://www.google.com`

## 不做的原因

1. 当前目标用户是开发者，手动导入 cookie 已够用
2. 实现复杂度高（Keychain/Secret Tool 调用 + SQLite + AES + 多浏览器适配）
3. 依赖 Chrome 内部存储格式，维护成本高（`meta.version >= 24` 的 hash 前缀就是一次格式变更）
4. 已有的 cookie 回写持久化机制（pi-web-access 没有的）减少了频繁导入的需求

## 如果未来要做

- 仅支持 macOS + Chrome 即可覆盖大部分用户
- 核心依赖: Go 标准库 `crypto/aes`, `crypto/cipher`, `database/sql` + `modernc.org/sqlite`
- 参考 pi-web-access 的 `chrome-cookies.ts`
