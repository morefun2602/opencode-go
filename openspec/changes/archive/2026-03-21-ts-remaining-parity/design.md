## Context

Go 版 OpenCode 已通过 `ts-parity-full-impl` 实现了核心骨架（ReAct 循环、OpenAI/Anthropic 双提供商、8 个内置工具、MCP 集成），但与上游 TypeScript 原版相比仍有显著功能缺口：提供商扩展性不足、缺少 4 个高频工具、无 Agent 模式系统、会话管理薄弱、TUI 停留在原始 REPL、事件通知与权限匹配能力缺失。

当前代码库状态：
- `internal/llm`：硬编码 openai / anthropic 两个提供商，无 registry
- `internal/tool` + `internal/tools`：8 个内置工具（read/write/edit/glob/grep/bash/task/webfetch），无标签/模式过滤
- `internal/runtime`：Engine 固定 build 模式，无模式切换
- `internal/store`：SQLite v3 schema，会话表无 title/archived/parent 列
- `internal/server`：基础 CRUD + complete 端点
- `internal/cli`：简单 REPL
- `internal/policy`：精确匹配的 ask/allow/deny

## Goals / Non-Goals

**Goals：**
- 实现 OpenAI-compatible 通用提供商与 provider registry，统一提供商管理
- 补齐 todowrite、apply_patch、websearch、question 四个工具
- 建立 Agent 模式系统（build/plan/explore + 自定义），按标签过滤工具
- 完善会话管理：fork、revert、title、archived、auto-summary、usage
- 引入进程内事件总线，暴露 SSE 端点
- 基于 Bubble Tea 构建 TUI，替代原始 REPL
- 升级权限系统支持 glob pattern 与 once/always/reject 语义
- 扩展 HTTP API 和配置系统

**Non-Goals：**
- 不实现全部 20+ 上游提供商（仅需 openai-compatible 兜底即可覆盖大部分）
- 不实现 LSP 集成
- 不实现 OAuth / 外部身份认证
- 不追求与上游 TypeScript TUI 完全一致的外观（保证核心交互功能对齐）
- 不引入前端 Web UI

## Decisions

### D1: Provider Registry 设计

**选择**：在 `internal/llm` 中新增 `registry.go`，维护 `map[string]ProviderFactory` 注册表。启动时根据配置遍历 `providers` 列表，按类型调用对应工厂函数（openai / anthropic / openai-compatible / stub）注册实例。

**替代方案**：
- 直接在 Engine 中 switch-case 硬编码 → 不可扩展
- 反射扫描 + 自动注册 → 过度设计

**理由**：显式注册表简单透明，新增提供商只需添加一个工厂函数和配置条目。

### D2: OpenAI-Compatible 提供商

**选择**：复用 `openai-go` SDK，仅在初始化时覆盖 `option.WithBaseURL(cfg.BaseURL)` 即可接入任何兼容 API。

**替代方案**：
- 裸 HTTP 客户端 → 重复大量 SDK 已有的序列化/错误处理代码

**理由**：SDK 已处理 SSE 解析、重试、类型安全，覆盖 BaseURL 是最小成本方案。

### D3: 工具标签与模式过滤

**选择**：在 `tool.Definition` 中新增 `Tags []string` 字段。每种 Agent 模式定义允许的标签集合。Engine 的 `collectTools` 方法在返回工具列表前按当前模式过滤。

**替代方案**：
- 按工具名硬编码白名单 → 不支持自定义模式和新工具
- 每个模式独立维护工具列表 → 维护成本高

**理由**：标签是灵活的分类机制，可自然扩展到 MCP 工具和自定义模式。

### D4: 事件总线

**选择**：在 `internal/bus` 中实现基于 Go channel 的简单 pub/sub。使用泛型 `Bus[T]` 或按类型区分的 `Publish(topic, payload)` / `Subscribe(topic) <-chan Event`。

**替代方案**：
- 使用第三方库（如 watermill）→ 过重，进程内不需要持久化和消息队列
- 直接回调注册 → 缺少解耦和并发安全

**理由**：进程内 channel-based pub/sub 足够简单、零外部依赖、天然并发安全。

### D5: TUI 架构

**选择**：基于 Bubble Tea 的 Elm 架构。顶层 Model 管理子视图（chat、sidebar、input、dialog），通过 Msg 驱动状态更新。Markdown 渲染使用 glamour，样式使用 lipgloss。

**替代方案**：
- 继续使用原始 REPL + readline → 无法满足上游对齐需求
- 使用 tview → 非 Elm 架构，生态不如 Charm 系列

**理由**：Charm 生态（bubbletea + lipgloss + glamour）是 Go 终端 UI 的事实标准，社区活跃，文档完善。

### D6: 会话 Fork / Revert

**选择**：Fork 在 SQLite 层面执行 `INSERT INTO messages SELECT ... WHERE session_id=? AND seq<=?`，创建新 session 行并关联 `parent_id`。Revert 执行 `DELETE FROM messages WHERE session_id=? AND seq>?`。两者均在事务内完成。

**替代方案**：
- 应用层逐条复制消息 → 慢且易出错

**理由**：批量 SQL 操作在同一事务内完成，原子性有保证且性能好。

### D7: 权限 Glob Pattern

**选择**：使用 `filepath.Match` 或 `path.Match` 进行 pattern 匹配。规则格式为 `tool_name:pattern`，匹配时先精确匹配 tool_name，再对关键参数（如 path）进行 glob 匹配。

**替代方案**：
- 正则表达式 → 对终端用户过于复杂

**理由**：Glob pattern 直觉简单，与 .gitignore 风格一致。

### D8: 数据库迁移 v3 → v4

**选择**：新增 `title TEXT DEFAULT ''`、`archived INTEGER DEFAULT 0`、`parent_id TEXT DEFAULT ''`、`parent_message_seq INTEGER DEFAULT 0` 列到 session 表。使用 ALTER TABLE ADD COLUMN。

**理由**：SQLite 支持 ALTER TABLE ADD COLUMN 且无需重建表，最小侵入性。

## Risks / Trade-offs

- **TUI 复杂度风险** → Bubble Tea 应用状态管理可能变复杂。通过严格的子视图分离和消息路由缓解。
- **工具数量增长** → 4 个新工具 + 标签系统增加注册逻辑。通过统一的 `RegisterTool` 函数和清晰的标签分类缓解。
- **apply_patch 正确性** → Unified diff 解析是已知的易错领域。考虑复用成熟的 `sourcegraph/go-diff` 库。
- **事件总线内存** → 慢消费者可能导致 channel 积压。通过有界 buffer 和丢弃策略缓解。
- **远程配置安全** → 从外部 URL 拉取配置存在 SSRF 风险。仅允许 HTTPS、限制响应大小、不合并敏感字段。
