# 架构演进方向计划 — 草案 v2

> **状态**: 草案，欢迎讨论。v2 收纳用户对 v1 的修正。
> **依据**: `har-20260524.md` 全部调研结论。
> **目标**: 把 HAR 中识别出的 28 个未实现 RPC 全部接入 library，CLI 按需暴露；同时把现有 6 个能跑的 RPC 在过程中**渐进迁移**到新架构下；同时为后续网络层优化预留干净的分离面。

## 1. 四条核心原则

### 1.1 渐进迁移（不是"旧的不动"）

- 不一次性大爆炸式重写，但也不让旧代码永久留在旧模式
- 做新 RPC 时，**同域的旧 RPC 一起迁过来**——例如做 `chat/meta.go` (MUAZcd) 时把 `chat.go` 里的 `MaZiqc`/`hNvQHb` 也迁到 `chat/list.go`/`chat/read.go`
- 每次迁移都是一个**完整功能域**的更新，到收尾时整个 `internal/client/` 都在统一架构下
- 单次迁移的爆炸半径 = 一个子包，可控

### 1.2 协议层 / 网络层 / 业务层严格分离

这是 v2 新增的核心原则，理由是当前 CLI 的体感速度明显落后浏览器，**后续必须做网络层优化（连接复用、并发 batch、流式解析等）**。要让那波优化不破坏协议正确性，三层从一开始就必须分开：

- **协议层** (`protocol/`): 纯函数，无 I/O，无状态。负责 payload encode 和 response decode。**100% 单元测试覆盖**
- **网络层** (`transport/`): HTTP 请求发送、SSE 解析、并发调度。**不知道 RPC 语义**，只看字节流
- **业务编排层** (`chat/`、`bootstrap.go` 等): 组合协议层 + 网络层，对外暴露 `c.ListChats(ctx, ...)` 这种语义化 API

本计划**不做**网络层优化，但**架构必须把它们分开**——这样未来做优化时，协议层不会被殃及。

### 1.3 Library 完整覆盖协议，CLI 选择性暴露

- Library: 覆盖**全部** RPC（含心跳/日志/写回），例外只有两类（见 §6.2）
- CLI: 只把对用户有价值的功能做成命令
- 二者解耦: library 实现 ≠ CLI 必须暴露

### 1.4 目录按虚拟页面组织（1:1 HAR 映射）

HAR §3 已经按 `source-path` 把 RPC 按虚拟页面分了组，直接映射到 Go 子包。

## 2. 三层架构详解

```
┌──────────────────────────────────────────────────────┐
│  业务编排层 (chat/, bootstrap.go, ...)               │
│  - 对外: c.ListChats(ctx, cursor) ([]ChatItem, ...)  │
│  - 对内: protocol.EncodeListChats() →                 │
│          transport.PostBatch() →                      │
│          protocol.DecodeListChats()                   │
└──────────────────┬───────────────────────────────────┘
                   │
       ┌───────────┴───────────┐
       ▼                       ▼
┌─────────────┐         ┌──────────────┐
│  协议层      │         │  网络层       │
│  protocol/  │         │  transport/  │
│             │         │              │
│  pure func: │         │  - HTTP POST │
│  - Encode   │         │  - SSE parse │
│  - Decode   │         │  - 并发调度  │
│             │         │  - 连接复用  │
│  无状态     │         │  - (未来)缓存│
│  无 I/O     │         │              │
│  全单测     │         │  不知 RPC 语义│
└─────────────┘         └──────────────┘
```

### 2.1 协议层 (`internal/client/protocol/`)

- **形态**: 纯 Go 函数。每个 RPC 一对 `Encode<Name>(args) (rpcID, payload string)` 和 `Decode<Name>(body []byte) (*Result, error)`
- **依赖**: 只依赖 `encoding/json`、`internal/types`。不依赖 net/http、context
- **测试**: 每个 RPC 必须有 baseline test，testdata 用真实 HAR response
- **注释**: 每个文件顶部有协议形状文档（见 §4）

### 2.2 网络层 (`internal/client/transport/`)

- **形态**: 接受协议层产出的 `(rpcID, payload)`，发送到 batchexecute，返回原始 body 字节流
- **职责**: HTTP 构造、headers、cookies、batchURL 拼接、SSE 帧解析、错误（HTTP 状态、网络错误）
- **不职责**: 不解析 RPC body 内容、不懂 wrb.fr envelope（这些归协议层）
- **未来优化空间**: 并发 batch、连接预热、HTTP/2 stream multiplexing、本地短期 cache

### 2.3 业务编排层（按虚拟页面组织的子包 + 顶层 facade）

- **形态**: 子包内一个文件对应一个 RPC 语义（如 `chat/meta.go`、`chat/turn.go`）
- **职责**: 调用协议层 encode → 喂给网络层 → 调用协议层 decode → 处理业务错误（重试、降级）
- **对 CLI 暴露**: 顶层 `internal/client` 包 re-export 关键方法，cmd 仍只 import `client`

## 3. 目标目录结构

```
internal/client/
  client.go               底座: Init + cookies + httpClient（已存在，部分会拆到 transport）
  rpc.go                  顶层 facade: 暴露 c.ListChats / c.GetUserProfile 等给 cmd
  
  protocol/               协议层（纯函数）
    envelope.go           wrb.fr 帧解析、length-prefixed frames、)]}'前缀
    encode.go             payload 编码共用工具
    types.go              协议层独立的轻量类型（不引 internal/types 避免循环）
    rpcs/
      o30o0e.go           GetUserProfile encode/decode + 注释
      o30o0e_test.go      baseline test
      k4wwud.go           GetUserLocation
      ...
    testdata/             真实 HAR response 落盘
      o30o0e_basic.json
      mazipc_paged.json
      ...
  
  transport/              网络层
    http.go               POST + headers + 错误分类
    batch.go              batchURL 构造 + form 编码 + reqID 自增
    stream.go             StreamGenerate / SSE 帧解析
    options.go            source-path、超时、bestEffort opts
  
  bootstrap.go            业务: 首屏并发 RPC（otAQ7b/GPRiHf/o30O0e/K4WWud/...）
  
  chat/                   业务: /app/<chatId>
    list.go               MaZiqc（迁移自旧 chat.go）
    read.go               hNvQHb（迁移自旧 chat.go）
    meta.go               MUAZcd（新）
    turn.go               EqPOKe（新）
    context.go            kwDCne（新）
    mark_read.go          k81mDb（新）
    delete.go             上游 GzXR5e + qWymEb（待 HAR）
  
  research/               业务: deep research 编排
    preflight.go          ESY5D / L5adhe / ku4Jyf / PCck7e（迁移自旧 research.go）
    create.go             编排逻辑（迁移）
    reports.go            jGArJ（新）
  
  quota/                  业务: qpEbW / aPya6c（迁移自旧 quota.go，整理双名）
  
  models/                 业务: otAQ7b 模型发现（迁移自旧 models.go）
  
  abuse/                  业务: GPRiHf（迁移自旧 abuse.go）
  
  upload/                 业务: resumable upload（迁移自旧 upload.go，拆出 transport 部分）
  
  generate/               业务: StreamGenerate 81-element + 三种模式（迁移自旧 generate.go）
                          注: 这是最敏感的，迁移最后做，且要严格保留 parse_test 覆盖
  
  daily_brief/            业务: /daily-brief
    stream.go             StreamGenerateYourDay（独立 81-element 之外的最小 builder）
  
  library/                业务: /library
  images/                 业务: /images
  notebook/               业务: /notebook/<id>
  gems/                   业务: 待 HAR
  housekeeping/           业务: library 实现、CLI 不暴露的心跳/日志/写回
```

迁移完成的标志: 原 `chat.go` / `research.go` / `quota.go` / `models.go` / `abuse.go` / `upload.go` / `generate.go` 这 7 个文件全部消失，内容分布在 protocol/ + transport/ + 业务子包。

## 4. 协议层代码注释规范

**这是 RPC 协议这种反向工程场景下的硬规范**——没有注释，下次修协议的人完全没法读懂。

每个 `protocol/rpcs/*.go` 文件**必须**包含：

```go
// Package-level comment block at file top, before any code:
//
// RPC: o30O0e — GetUserProfile
// Source-path: /app or / (any page)
// Reject codes: none observed in HAR 20260524
//
// Payload shape:
//   [["me"], [[["person.photo","person.name","person.email"]], null, [1,7]]]
//   ↑       ↑                                                  ↑    ↑
//   subject field list                                         ?    pagination?
//
// Response shape:
//   [["me", 1, ["<userId>", [null,...,[null,["me"]]], [profile_arr]]]]
//                ↑           ↑                       ↑
//                user id    metadata               profile data
//
//   profile_arr structure:
//     [0]: [true, 0, true, null, ..., "<userId>", ..., [unix_seconds, nanos]]
//     [1]: "<display name>"
//     [3]: "<email>"
//
// HAR sample: testdata/o30o0e_basic.json (captured 2026-05-24)
//
// Notes:
//   - "me" can also be a specific account id; HAR only uses "me"
//   - Field [1,7] in payload is unclear; HAR shows it constant; pass-through

package rpcs
```

最低要求：

- **RPC ID** 和**推测语义名称**
- **source-path 约定**（默认 `/app` / 必须带 chat id / 等）
- **payload shape**（注释里画出数组结构，每个 slot 含义）
- **response shape**（同上）
- **reject code 列表**（如果观察到）
- **HAR sample 指针**（指向 testdata 文件名）
- **不确定项标注**（哪些 slot 没搞清楚含义；让后人有线索）

业务层和网络层注释按 Go 常规规范，无需此模板。

## 5. RPC 抽象层 API（业务层使用）

业务层不直接调 `protocol` + `transport`，通过顶层 facade `rpc.go` 调用：

```go
// rpc.go (业务层 facade)

type RPCOpt func(*rpcConfig)

func WithSourcePath(sp string) RPCOpt
func WithSourceCid(cid string) RPCOpt   // 简写: <appPath>/<cid>
func WithBestEffort() RPCOpt
func WithTimeout(d time.Duration) RPCOpt

// 单次 RPC
func (c *Client) CallRPC(ctx context.Context, rpcID, payload string, opts ...RPCOpt) (body []byte, rejectCode int, err error)

// 并发批量（同一 batchexecute 里多个 RPC，HAR 中常见）
type RPCCall struct{ ID, Payload string }
func (c *Client) CallRPCBatch(ctx context.Context, calls []RPCCall, opts ...RPCOpt) (bodies map[string][]byte, err error)
```

**业务层典型调用**:

```go
// chat/meta.go
func (c *Client) GetChatMetadata(ctx context.Context, chatID string) (*ChatMeta, error) {
    rpcID, payload := protocol.EncodeGetChatMetadata(chatID)
    body, _, err := c.CallRPC(ctx, rpcID, payload, WithSourceCid(chatID))
    if err != nil { return nil, err }
    return protocol.DecodeGetChatMetadata(body)
}
```

注意三层职责清晰: encode 在 protocol、HTTP 在 transport（被 CallRPC 调用）、组合在业务层。

## 6. 实现 / 暴露分层

### 6.1 Library 必须实现

| 分类 | RPC | 备注 |
|---|---|---|
| **功能 A 类** | kwDCne EqPOKe MUAZcd jGArJ cYRIkd uPDUsc MyzX6c mhs1xe NXpLKc o30O0e K4WWud XhaU0b V8rlHe Pty9pd | HAR §2.2 A 类 14 个 |
| **写回 / 心跳 B 类** | CNgdBe Te6DCf maGuAc k81mDb Ub3MPb dI8W6e TFNzk ozz5Z VxUbXb | HAR §2.2 B 类 9 个 |
| **C 类（部分）** | sJBwce P3BxXb | UI-only 但 library 可有 |
| **新端点** | StreamGenerateYourDay | 独立 builder |
| **上游待 HAR** | GzXR5e qWymEb / oMH3Zd kHv0Vd UXcSJb / c8o8Fe | 抓 HAR 后实现 |
| **旧 RPC 迁移** | MaZiqc hNvQHb otAQ7b GPRiHf ESY5D L5adhe ku4Jyf qpEbW aPya6c PCck7e | 现已实现，需迁到新结构 |

### 6.2 Library 不实现（仅两类）

| RPC / 端点 | 不实现理由 |
|---|---|
| `waa-pa/.../Waa/Create` | HAR §9.2 调研确认 token 无后续引用，纯遥测 |
| `signaler-pa/punctual/*` | long-polling，实现成本远高于其他 RPC，CLI 场景用不上 |

### 6.3 Protobuf 暂缓

| RPC | 状态 | 何时考虑 |
|---|---|---|
| `HcT8bb` GetNotebook | base64 protobuf | 等真要做 NotebookLM 写流程时一起 |
| `LyXzt` MyActivity log | 17KB base64 protobuf | 不计划做 |

### 6.4 CLI 暴露建议

| 子命令 | 对应 library 模块 |
|---|---|
| `status` 增强 | bootstrap.go 全部 |
| `day` | daily_brief/ |
| `chat meta/turn/delete` | chat/ |
| `research list` | research/reports.go |
| `images templates/discover` | images/ |
| `notebook list` | notebook/ |
| `gems list/create/edit/delete` | gems/（待 HAR） |

CLI **不暴露**: housekeeping/ 全部、bootstrap 中部分细粒度 RPC（如 MyzX6c 只通过 status 间接出现）。

## 7. 执行顺序（草案）

按"先打地基、再加新 RPC、再迁老 RPC"组织，**新接每个域时把旧 RPC 一并迁过去**。

| 阶段 | 内容 | 阻塞 | 风险等级 |
|---|---|---|---|
| **P0** | 建协议层底座: `protocol/envelope.go` (wrb.fr / length frame / )]}'前缀) + 至少 3 个旧 RPC 的 protocol 实现 + 完整 baseline test。**只新增**，旧调用路径不动 | 无 | 低（纯新增） |
| **P1** | 建网络层底座: `transport/http.go` + `transport/batch.go`，从 `client.go` 抽出（但 `client.go` 旧函数仍保留 façade 转发） | P0 | 低（旧调用路径仍可走） |
| **P2** | 顶层 facade `rpc.go`: `CallRPC` / `CallRPCBatch` API，第一个 RPC `o30O0e` 走完完整三层流程 | P0 P1 | 低（纯新增） |
| **P3** | bootstrap 余下 RPC: K4WWud / cYRIkd / uPDUsc / MyzX6c / mhs1xe | P2 | 低 |
| **P4** | `status` 命令集成 P3 数据 | P3 | 低（只改 cmd/status.go） |
| **P5** | `daily_brief/` + `day` 命令（独立 builder，不触 generate.go） | P2 | 低 |
| **P6** | `chat/` 子包: 新增 MUAZcd/EqPOKe/kwDCne/k81mDb + **迁移旧 MaZiqc/hNvQHb**。迁移完删除旧 chat.go | P2 | 中（迁移旧代码，但 parse_test.go 兜底） |
| **P7** | `research/` 子包: 迁移整个 deep research 流程 + 新增 jGArJ | P2 | 中（preflight 编排较复杂） |
| **P8** | `quota/` `models/` `abuse/` 迁移 — **Stage 5 completed models/quota; abuse remains upstream PR #310 follow-up** | P2 | 低 |
| **P9** | `images/` 子包 | P2 | 中（需要确认 generate.go 是否要接 template id） |
| **P10** | `notebook/` 子包（不含 HcT8bb protobuf） | P2 | 低 |
| **P11** | `upload/` 迁移 + `generate/` 迁移 (StreamGenerate) — **Stage 5 completed generate; upload remains future stage** | P2 | 高（generate.go 是协议热点）|
| **P12** | `gems/` + 上游 RPC 验证 | 阻塞：缺 HAR | — |
| **(独立)** | 命名修正: L5adhe (→PrefsSync) / ESY5D (→BardSettings) / qpEbW+aPya6c 双名整理 | 在 P6/P7/P8 顺手完成 | 低 |
| **(独立)** | `housekeeping/` 子包（B 类大部分） — **Stage 5 completed debug-triggerable encoders** | P2 | 低 |

**可选并行**: P3-P10 互不依赖，做完 P0-P2 后可任意排列。P11 留在最后单独做。

## 8. 风险表

| 风险 | 严重度 | 缓解 |
|---|---|---|
| 旧 RPC 迁移引入回归 | 中-高 | 协议层 100% 单元测试（用旧 parse_test.go 现有 testdata + 新增 HAR 真实样本）；迁移前后必须保持现有 prek/test 通过 |
| 协议层和网络层职责蔓延 | 中 | 协议层禁止 import `net/http`/`context`；CI 加 lint 检查 |
| 新 RPC 响应 schema 漂移 | 中 | 每个 RPC 必须配 testdata baseline；协议形状用注释固化 |
| `StreamGenerate` 81-element 漂移 | 高 | 留到 P11 最后做；StreamGenerateYourDay 独立 builder |
| 网络层未来优化破坏协议正确性 | 中 | 三层严格分离，协议层有完整单元测试，网络层优化时只跑协议测试就能初步验证 |
| rate limit / 滥用检测 | 中 | CLI 按需触发；housekeeping 只是 library 备用 |
| 命名修正破坏向后兼容 | 低 | 仅改 const 名，不动调用关系；grep + 测试 |
| 子包化造成 import cycle | 低 | 协议层独立 types.go，业务层 import 协议层；不反向 |
| protobuf 阻塞 NotebookLM | 中 | 先做 NXpLKc list；HcT8bb 等真要做完整流程时再处理 |

## 9. 待讨论 / 决策点

1. **旧 RPC 迁移节奏: 顺手迁 vs 集中批量迁？**
   - 顺手迁: 做新 RPC 时把同域旧 RPC 一起迁（v1 即此模式）
   - 集中迁: 先把所有新 RPC 接完，最后一次性迁旧的
   - 倾向: 顺手迁——心智负担小，每次迁移范围可控

2. **`housekeeping/` 子包要不要现在就做？**
   - 做: library 完整性强
   - 不做: YAGNI，等真有需求再加
   - 倾向: 等到第一个需要嵌入心跳/写回的功能出现时再做

3. **NotebookLM 域要不要进入 protobuf 解码？**
   - 短期不做；评估完产品价值后再说

4. **HAR 缺失场景补抓优先级？**
   - 建议先抓: Gems CRUD + DeleteChat
   - 其他按需

5. **命名修正什么时候做？**
   - 选项 A: P0 之前先做
   - 选项 B: 在 P6/P7/P8 迁移对应模块时顺手做
   - 倾向: B（避免单独的"rename"提交污染历史；迁移本来就要碰这些 const）

6. **`internal/client` 子包对 cmd 的暴露形式？**
   - 方案 A: 顶层 `client` 包 re-export 所有方法（facade 模式），cmd 仍只 import `client`
   - 方案 B: cmd 直接 import `client/chat`、`client/research` 等子包
   - 倾向: A——保持 cmd 简洁；facade 也是天然的"哪些 API 对外暴露"的总览

7. **网络层优化什么时候做？**
   - 本计划不做，但需要确认: 三层分离的架构出来后，下一波是不是直接接网络优化？
   - 倾向: 三层分离 + 全 RPC 接入完成后再启动网络优化专项

8. **`signaler-pa` 完全不做 vs 留 placeholder？**
   - 倾向: 不做。deep research 等流程目前是轮询，已经能 work

## 10. 验证 / 完成标准

每个 P 完成需满足:

1. 对应 RPC 在 `internal/client/protocol/rpcs/*.go` 有 encode/decode 实现，**含 §4 规定的注释块**
2. 对应 testdata 落盘到 `internal/client/protocol/testdata/`
3. 单元测试覆盖该 RPC 的 encode/decode 正反向
4. 业务层包装函数有，对外通过 `rpc.go` facade 暴露
5. 如果暴露到 CLI，对应命令通过 `make build` + 手动跑通
6. 不引入对现有命令的回归: `prek` 通过 + 现有 `ask`/`reply`/`list`/`research`/`status` 跑通

全部完成标志:

- HAR 报告 §2.2 的 28 个未实现 RPC，library 覆盖率 ≥ 25（除 Waa + signaler + 2 个 protobuf）
- 现有 10 个已实现 RPC 全部迁到新结构，旧的 7 个文件删除
- HAR 报告 §4 缺失场景至少补抓 3 个并接入
- 命名修正全部完成
- 协议层单元测试覆盖率 100%（按 RPC 数量算）
- `status` 命令显示账户/能力/配额/工具完整面板
