# Design：与上游 TypeScript OpenCode 对齐（架构与落地策略）

## Context

- **背景**：见 `proposal.md`。当前 Go 代码树以 `cmd/opencode`、`internal/{config,cli,runtime,llm,tools,store,server}` 为主，行为面远小于上游 `packages/opencode`（Effect 运行时、Hono 控制面、Drizzle 存储、MCP SDK、丰富 `src/tool/*`、skill、plugin、ACP 等）。
- **约束**：须保持 `internal` 边界与模块 `github.com/morefun2602/opencode-go`；与主规格 delta 一致；**不**要求目录与 TS 一一对应，但**集成点与语义**需可对齐或显式记录差异。
- **干系人**：本仓库维护者；未来消费 HTTP/CLI 的编辑器与自动化。

## Goals / Non-Goals

**Goals:**

- 给出**分层架构**：MCP、内置工具、技能、插件、ACP 与现有 **runtime / llm / server / store** 的依赖方向与接口边界。
- 固定**阶段性策略**：先打通「可测、可扩展」的骨架（注册、调用、错误语义），再按任务清单填充与上游同名的工具与协议细节。
- 约定**持久化与上游 schema 的映射原则**（逻辑实体与版本迁移），具体列名在实现与 spec 中交叉引用，避免在 design 中写死未经验证的每一列。
- 约定 **HTTP/CLI** 扩展方式：新路由与子命令如何接入同一配置与鉴权基线。

**Non-Goals:**

- 完整复刻 **桌面/Web/Console** 包或上游 UI。
- 在本文件中锁定**每一个**第三方库版本或具体函数签名。
- 替代上游 TypeScript 实现的发布与分发渠道。

## Decisions

### 1. 分层与依赖方向（自上而下）

- **决策**：调用链保持 **单向**：`cli` / `server` → `runtime`（会话编排）→ `llm`（模型）与 **Tool/MCP 门面** → `store`（持久化）。**禁止** `store` 或底层工具直接依赖 HTTP handler。
- **MCP（`internal/mcp` 或等价）**：实现 **客户端** 连接外部 MCP 服务；**宿主**侧将 MCP 工具表注册进统一 `ToolRouter`（名称与上游可配置前缀一致，避免与内置工具冲突）。与 `builtin-tools` 共享同一「调用-结果-回注模型」接口。
- **内置工具（`internal/tool` 或 `internal/builtin`）**：每个工具实现小接口（名称、JSON schema、执行 `context.Context`），由 **权限/策略** 层在调用前裁决（对齐上游「需确认」语义，细节在 `builtin-tools` spec）。
- **技能（`internal/skill`）**：负责发现、解析元数据、**注入**到 prompt 或工具列表；不直接执行 shell，执行仍走工具/MCP。
- **插件（`internal/plugin`）**：采用 **进程内** 动态加载（`plugin` 包）或 **独立 RPC** 二选一；首版设计倾向 **进程内 + 显式接口**，降低运维复杂度；若选 RPC，在实现任务中单开变更。
- **ACP（`internal/acp`）**：作为 **适配层**，将 ACP 事件/会话映射到现有 `runtime` 会话模型；不复制上游 Effect 运行时。

### 2. 与上游 TS 模块的语义对齐方式

- **决策**：以 **能力规格** 为真源；每个能力在文档中维护 **「上游参照」**（例如 `packages/opencode/src/tool/bash.ts` 的行为摘要），**不**把 TS 源码当作编译依赖。
- **备选**：子模块 git submodule 指向 opencode — **否决**（绑定发布节奏）。

### 3. HTTP 控制面扩展

- **决策**：在现有 `/v1` 版本前缀下**增量**增加 REST 资源（会话列表、消息分页、流式端点）；**BREAKING** 时升 `/v2` 或新前缀。OpenAPI 文档可滞后于端点，但须在 spec 中要求「每个新端点有稳定错误体」。
- **备选**：另起独立端口 — **否决**（增加运维与鉴权重复配置）。

### 4. LLM 与工具链路的划分

- **决策**：`llm` 包仅负责 **模型 I/O**；**工具调用循环**（tool_calls 解析、并行/串行策略）放在 `runtime` 或专用 `orchestrator` 子包，避免 `llm` 依赖 `tool` 的具体实现。
- **备选**：全部挤在 `llm` — **否决**（循环依赖风险）。

### 5. 持久化映射

- **决策**：以上游 **逻辑实体**（会话、消息、turn、附件元数据等）为锚点，在 Go 侧用迁移版本递增；**表结构**可与 Drizzle 不完全同名，但需维护 **映射表**（文档 + 测试）。跨大版本迁移须离线备份提示（与现有 `PRAGMA user_version` 策略一致）。
- **备选**：直接复制 Drizzle SQL — **否决**（SQLite 方言与历史包袱不同）。

### 6. 安全与沙箱

- **决策**：`bash`/进程类工具默认 **最小权限**（工作区根、超时、输出上限）；高危操作必须经 **policy** 与审计日志。具体阈值在 `builtin-tools` spec 中量化。
- **备选**：无限制执行 — **否决**。

### 7. 插件机制（首版）

- **决策**：**不**将「纯 Go `plugin` 动态库」作为首版唯一路径。首版采用 **编译期注册 / 静态链接扩展**（与主二进制一起发布），或 **子进程 + stdio/JSON-RPC** 等可跨平台方案；保证 **Windows** 与 **linux/darwin** 同一套用户故事可跑通。
- **理由**：`plugin` 包**不支持 Windows**，且要求主程序与插件同 Go 版本、同构建方式，发布与 CI 成本高。
- **备选**：首版即上 `plugin`（仅 linux/darwin）— **否决** 作为默认；若未来提供，可作为 **可选** 构建 tag，文档标注平台限制。

### 8. ACP 与 HTTP

- **决策**：ACP 与现有 **HTTP 服务共用同一监听地址与端口**，并复用 **同一套鉴权中间件**（Bearer / 共享密钥等与 `http-api`、`cli-and-config` 一致）；ACP 路由挂在 **同一 `http.Server`** 下（例如 `/v1/acp/...` 或上游约定路径，具体由 `acp-bridge` spec 固定）。
- **理由**：避免第二套端口、第二套凭据与运维面；与 `design` 中「单一控制面」一致。
- **备选**：独立本地 socket — **保留为后续**（嵌入式 IDE 或特权场景再评估）。

## Risks / Trade-offs

- **[Risk]** 上游快速迭代导致对齐文档过期 → **Mitigation**：`upstream_compat_ref` 与定期人工 diff；关键行为用契约测试。
- **[Risk]** MCP 与内置工具命名冲突 → **Mitigation**：统一注册表与命名空间（内置前缀 `opencode.` 或配置化）。
- **[Risk]** 插件接口不稳定 → **Mitigation**：首版缩小表面（仅 hooks），或标记 **experimental**。
- **[Trade-off]** 进程内插件 vs 安全隔离 → 首版优先可交付；强隔离放到后续变更。

## Migration Plan

- **数据**：新增表/列通过版本化迁移；回滚策略为「降级二进制 + 保留备份」；若迁移仅向前，在发行说明中写明。
- **API**：新端点默认不删除旧端点；**BREAKING** 需双写期或版本前缀。
- **配置**：新键使用顶层字段并文档化，避免静默覆盖。

## Open Questions

- 与上游 **消息模型 v2** 的字段级对齐是否在一期完成，还是分「读兼容 / 写兼容」两阶段（由 `agent-runtime` spec 拍板）。
