# Design: OpenCode Go 实现

## Context

- **背景**：变更提案（`proposal.md`）要求在本仓库建立与上游 OpenCode **功能域对等**的 Go 实现，强调模块边界、可测试性与可发布二进制，而非逐句翻译。
- **现状**：`opencode-go` 以 OpenSpec 与 Cursor 工作流为主，**尚无 Go 业务代码**；无存量用户数据或线上服务需无缝迁移。
- **约束**：需与后续 `specs/*/spec.md` 对齐；实现阶段应保持 `internal` 包边界清晰，避免“大泥球”单包。
- **干系人**：本仓库维护者、未来接入内部发布/CI 的平台团队（发布细节在任务阶段落地）。

## Goals / Non-Goals

**Goals:**

- 定义 **Go 代码树与分层**（`cmd`、`internal`、可选 `pkg`），使 **CLI、HTTP API、**运行时装配、LLM/工具、持久化可独立演进与测试。
- 固定 **横切关注点**：配置加载顺序、日志、上下文取消、错误包装与退出码策略，与 `cli-and-config` spec 一致。
- 记录 **关键技术选型及备选方案**，便于评审与后续替换（例如 SQLite 驱动、HTTP 栈）。

**Non-Goals:**

- 在本文件中锁定**每一个**第三方库 minor 版本或具体 API 形状（留待实现与 `go.mod`）。
- 规定上游 TypeScript 仓库的目录一一映射关系。
- 详细序列图或逐接口的 protobuf/OpenAPI（若需要，在对应 spec 或单独附录中补充）。

## Decisions

### 1. 模块与目录布局

- **模块路径**：`github.com/morefun2602/opencode-go` — 与仓库根目录 `go.mod` 的 `module` 指令一致；对外 import 与模块缓存路径均使用该前缀。
- **决策**：单 Go module 根置于仓库根目录（或显式子目录 `go/`，若未来与多语言共存再拆分）；入口为 `cmd/opencode`（名称可随产品名调整），业务逻辑放在 `internal/` 下按域分包，例如：
  - `internal/config` — 配置模型与解析
  - `internal/cli` — 子命令、flag、与 core 的粘合（保持薄）
  - `internal/runtime` 或 `internal/session` — 会话/任务生命周期（与 `agent-runtime` spec 对应）
  - `internal/llm` — 提供商抽象与流式接口
  - `internal/tools` — 工具/MCP 调度与结果回注
  - `internal/store` — 持久化实现细节对上层隐藏接口
  - `internal/server`（或 `internal/httpapi`）— **首版即交付**的 HTTP 服务：路由注册、`http.Server` 生命周期、中间件（请求 ID、鉴权钩子），**不**承载 LLM 提供商出站细节（仍属 `internal/llm`）
- **理由**：符合 Go 社区常见惯例，`internal` 强制编译期边界，避免外部项目错误依赖实现细节。
- **备选**：多 module（微仓库）——对本阶段过重，增加版本协调成本，**否决**。

### 2. 配置：文件 + 环境变量 + 优先级

- **决策**：采用显式优先级（例如：**环境变量覆盖配置文件**，CLI flag 覆盖环境变量——具体字段在 `cli-and-config` spec 中列出）；配置模型用 Go struct + 校验函数（如 `Validate()`），启动失败时非零退出。
- **与上游对齐（已拍板）**：**配置文件名、文件格式（若与上游一致）、以及 JSON/YAML 键名**须与 **上游 OpenCode** 保持一致，以便同一工作区配置可被两种实现共用、文档与示例可复用。具体文件名、键清单与默认值以 `cli-and-config` spec 为准（实现时可引用或同步上游文档/ schema）。若 Go 侧需要**仅本实现**可用的项，须使用**上游未占用的键**或经约定的命名空间前缀，并在 spec 中列出，避免静默覆盖上游语义。
- **理由**：可重复构建与运维可观测性优于隐式魔法；与十二因子兼容；与上游共享键名降低迁移与心智成本。
- **备选**：仅环境变量——对本地开发与 IDE 集成不友好；**备选保留为测试场景**。

### 3. 日志与可观测性

- **决策**：默认使用标准库 **`log/slog`**（Go 1.21+）结构化日志；日志级别与 key 命名在代码审查中统一，关键路径带 `request/session` 等字段（与 spec 中的可观察需求一致）。
- **理由**：零额外大依赖、与 `testing` 输出易衔接。
- **备选**：`zap`/`zerolog`——性能更优但引入依赖；若后续证明热点在日志，可局部替换并在此设计增补决策。

### 4. 错误与退出码

- **决策**：库代码返回 `error`，边界使用 `fmt.Errorf("...: %w", err)`；CLI 层将**已分类错误**映射到稳定退出码（例如：用法错误 2、内部错误 1、用户取消 130——具体表在 `cli-and-config` spec）。
- **理由**：便于脚本化与 CI。
- **备选**：全程 panic — **否决**。

### 5. LLM 出站 HTTP（调用模型供应商）

- **决策**：以 **`net/http`** + 标准 `context` 实现**出站**提供商客户端；流式响应用 `io.Reader`/Scanner 或按上游 API 拆分为 chunk 迭代器，上层 `runtime` 只消费统一接口。
- **理由**：依赖面小，易测试（`httptest`）。
- **备选**：生成式 OpenAPI 客户端——可在 provider 稳定后引入，**不阻塞首版**。

### 6. HTTP API 服务（入站，首版必备）

- **决策**：首版即提供 **入站 HTTP API**（与 CLI 并存或作为 `serve`/`daemon` 子命令启动，具体交互在 `cli-and-config` spec 中固定），供编辑器与远程客户端调用；实现基于标准库 **`net/http`**（或 `x/net/http2` 仅在有明确需求时引入），路由与 handler 放在 `internal/server`，通过**窄接口**依赖 `runtime`/`store`，避免 handler 直接访问全局状态。
- **契约**：REST 或 JSON-RPC 等风格在 **`http-api` spec** 中定义（含版本前缀如 `/v1`、SSE/WebSocket 是否首版包含等）；与 `agent-runtime` 的可观察行为对齐。
- **监听与默认安全**：默认绑定 **loopback**（如 `127.0.0.1`）以降低首版暴露面；若配置为 `0.0.0.0` 或非本机访问，**必须**配合鉴权（见下）并在文档中醒目标注。
- **鉴权基线**：首版至少支持一种可自动化配置的方式（例如：**静态 Bearer token** 或 **从环境/文件读取的共享密钥**）；TLS 终止可首版依赖反向代理，或在 spec 中要求可选 `tls_cert`/`tls_key` 自服务。
- **理由**：与提案中 `http-api` 能力一致；与 Decision 5 的出站 HTTP 分离，避免混淆。
- **备选**：首版仅 gRPC — 增加客户端与调试成本，**否决**。

### 7. SQLite 与持久化

- **决策**：默认 **SQLite** 作为本地存储时，优先评估 **`modernc.org/sqlite`**（纯 Go）以满足交叉编译与无 CGO 流水线；若团队强制 CGO 或性能不达标，再评估 `mattn/go-sqlite3`。
- **理由**：与 `persistence` spec 中的迁移、完整性需求一致；选型差异应被 `store` 接口隔离。
- **备选**：嵌入式 KV（如 bbolt）——若 spec 要求关系查询与迁移，SQLite 更贴切。

### 8. 并发与取消

- **决策**：长任务与 LLM 调用全程传递 **`context.Context`**；会话级关闭应级联取消子操作。
- **理由**：与 Go 惯例一致，便于测试超时。
- **备选**：全局 channel 管理——易泄漏，**否决**。

## Risks / Trade-offs

- **[Risk]** 与上游行为细微不一致导致用户困惑 → **Mitigation**：在对应 spec 中写清“对齐策略”；对关键路径加集成测试或契约测试。
- **[Risk]** 纯 Go SQLite 与 CGO 版本性能/兼容性差异 → **Mitigation**：`store` 接口后切换实现；CI 矩阵覆盖目标平台。
- **[Risk]** LLM 提供商 API 频繁变动 → **Mitigation**：`internal/llm` 按提供商分包，版本化适配层；失败语义在 `llm-and-tools` spec 中固定。
- **[Risk]** HTTP API 误暴露或弱鉴权 → **Mitigation**：默认 loopback、强制非本机监听时的鉴权、在 `http-api` spec 中写清威胁模型与测试用例。
- **[Risk]** 上游调整配置 schema 导致本实现解析失败或静默忽略字段 → **Mitigation**：在 `cli-and-config` spec 中约定**跟踪的上游版本/兼容性**；对未知键可记录警告；重大变更在发布说明与迁移任务中显式列出。
- **[Trade-off]** 强 `internal` 边界 vs 复用库化代码 → 若未来需被其他 Go 应用导入，再将稳定 API 抽到 `pkg/` 并 semver。

## Migration Plan

- **部署**：首版为**新代码库/新二进制**，无强制数据迁移；若日后提供从上游格式的导入，在 `persistence`/`cli` spec 中单独增需求与任务。
- **回滚**：发布失败时回退到上一 tag 二进制；数据库文件由版本化迁移支持向前兼容，**回滚二进制**时若迁移已执行，需在任务阶段定义“迁移版本门槛”与用户提示。
- **团队**：先合并设计与 specs，再按 `tasks.md` 分阶段实现，避免无 spec 的大块编码。

## Open Questions

- **HTTP API 细节**（REST 路径表、是否首版含 SSE、与 CLI 的启动组合方式）在 `http-api` / `cli-and-config` spec 中定稿；本设计仅锁定“首版必有 + 安全基线”。
