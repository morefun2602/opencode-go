## Context

opencode-go 在第一轮同步后已实现了大量基础模块，但这些模块多数处于「已实现但未接入」状态（Snapshot、LSP、FileWatcher），且缺少 Agent 框架的核心编排层（系统提示构建、模型路由、Agent 定义、Compaction 会话集成）。TypeScript 版本在这些方面有成熟实现，本次设计以 TS 版本为参考，结合 Go 的工程实践做出技术决策。

按 AGENTS.md 约束：仅需支持 OpenAI 和 Anthropic 两家 Provider，使用其 Go SDK。

## Goals / Non-Goals

**Goals:**

- 构建完整的系统提示编排链（模型 base prompt → 环境信息 → InstructionPrompt → 技能列表）
- 实现 Provider 路由和模型选择（default model / small model / parse model）
- 定义 7 种 Agent（build/plan/explore/general/compaction/title/summary）及其工具权限
- 将 Compaction 真正集成到 ReAct 会话流程（isOverflow → prune → process）
- 实现统一工具输出截断服务
- 接入已实现但未 wire 的模块（Snapshot、LSP、FileWatcher）
- 增强 read 工具和配置系统

**Non-Goals:**

- 不增加 OpenAI/Anthropic 之外的 Provider 支持
- 不实现动态插件加载（保持编译期静态 Hook）
- 不实现 Web/Desktop UI（仅 TUI + HTTP API）
- 不实现 models.dev 外部模型源集成
- 不实现文件保护（Protected paths）模块（优先级较低）

## Decisions

### D1: 系统提示构建 — 新建 internal/prompt/ 包

**选择**：新建 `internal/prompt/` 包，统一管理系统提示的构建。提示按层叠加：`ModelPrompt → AgentPrompt → Environment → InstructionPrompt → SkillSummary`。

**替代方案**：在 Engine 内联拼接 → 职责混乱，难以测试和扩展。

**细节**：
- `ModelPrompt(providerType string) string`：按 provider 类型返回基础 prompt（仅 OpenAI 和 Anthropic 两套）
- `EnvironmentPrompt(workspaceRoot string) string`：注入工作目录、平台、日期、git 分支/状态
- `InstructionPrompt(workspaceRoot string, configInstructions []string) string`：向上查找 AGENTS.md / CLAUDE.md / CONTEXT.md，加载 config.instructions 中的文件路径和 URL
- `SkillSummary(skills []skill.Skill) string`：仅注入技能名称和描述列表（不含完整正文），正文通过 skill 工具按需加载
- `Build(opts BuildOpts) string`：组装最终系统提示

### D2: Provider 路由 — 扩展 internal/llm/ 包

**选择**：在 `internal/llm/` 中新增 `routing.go`，实现模型选择逻辑。Engine 从单一 Provider 改为通过 Router 按需获取 Provider+Model。

**替代方案**：在 Engine 中硬编码模型选择 → 不可扩展。

**细节**：
- `ModelRef{ProviderID, ModelID string}`：模型引用
- `ParseModel(s string) ModelRef`：解析 `"provider/model"` 格式
- `Router.DefaultModel() ModelRef`：config.model → 第一个可用 provider 的第一个模型
- `Router.SmallModel() ModelRef`：config.small_model → 按 provider 优先级选择小模型（Anthropic: claude-haiku 系列，OpenAI: gpt-4o-mini 系列）
- `Router.Resolve(ref ModelRef) (Provider, string)`：返回 Provider 实例和 modelID
- Engine 的 `LLM` 字段改为 `Router *llm.Router`，每次调用时通过 Router 获取 Provider

### D3: Agent 定义 — 新增 internal/runtime/agent.go

**选择**：定义 `Agent` 结构体，包含 Name、Prompt、ToolPermissions（allow/deny 规则），替代当前简单的 Mode 结构体。保留 Mode 作为 Agent 的一个属性。

**替代方案**：仅扩展 Mode → 无法支持 Agent 级 prompt 和细粒度权限。

**细节**：
- 7 个内置 Agent：
  - `build`：默认，全部工具，无自定义 prompt
  - `plan`：禁止 write 标签工具，无自定义 prompt
  - `explore`：仅 read 标签工具，有专用 prompt（强调 glob/grep/read）
  - `general`：子 agent 模式，禁止 todoread/todowrite
  - `compaction`：隐藏，无工具，有专用 prompt（总结对话）
  - `title`：隐藏，无工具，有专用 prompt（生成 ≤50 字符标题）
  - `summary`：隐藏，无工具，有专用 prompt（生成 2-3 句摘要）
- `ToolFilter(agent Agent, allTools []ToolDef) []ToolDef`：根据 Agent 的权限规则过滤工具列表
- 隐藏 Agent 不出现在用户可选列表中，仅内部使用

### D4: Compaction 会话集成 — 重构 Engine + Compactor

**选择**：将现有 Compactor 重构为完整的三阶段流程（isOverflow → prune → process），集成到 ReAct 循环中。使用 compaction agent 而非通用 prompt 生成摘要。

**替代方案**：保持独立 Compactor 不集成 → token 溢出时 ReAct 循环直接失败。

**细节**：
- `IsOverflow(tokenUsage Usage, modelLimit int) bool`：基于 token 使用量和模型上下文限制判断
- `Prune(msgs []Message, keepRecentTokens int) []Message`：从旧到新裁剪 tool_result 内容（保留最近约 40k tokens），跳过 skill 工具结果
- `Process(ctx, provider, msgs) ([]Message, error)`：使用 compaction agent 调用 LLM 生成摘要，替换旧消息
- 触发时机：每轮结束后检查 isOverflow；Provider 返回 ContextOverflow 时触发
- 配置：`config.compaction.auto`（默认 true）、`config.compaction.reserved`（预留 token，默认 20000）、`config.compaction.prune`（默认 true）

### D5: Provider 消息转换 — 新增 internal/llm/transform.go

**选择**：在 `internal/llm/` 新增 `transform.go`，实现 per-provider 消息规范化。仅覆盖 OpenAI 和 Anthropic 两家的差异。

**替代方案**：不做转换 → Anthropic 对空 content 和 toolCallId 格式敏感，可能导致 API 错误。

**细节**：
- `TransformMessages(msgs []Message, providerType string) []Message`：
  - Anthropic：过滤空 text content、确保 toolCallId 仅含 `[a-zA-Z0-9_-]`
  - OpenAI：无特殊转换（SDK 已处理）
- 在 Provider.Chat/ChatStream 内部调用，对上层透明

### D6: 统一截断服务 — 新建 internal/truncate/ 包

**选择**：新建 `internal/truncate/` 包提供统一截断逻辑。所有工具通过 Registry 层统一接入。

**替代方案**：各工具自行截断 → 行为不一致，维护困难。

**细节**：
- `Truncate(output string, opts Options) Result`
- `Options{MaxLines int, MaxBytes int, Direction Direction}`
- `Direction`：Head / Tail
- 默认 `MaxLines=2000, MaxBytes=50*1024`
- `Result{Output string, Truncated bool}`
- 在 `tools.Registry.Run()` 返回后统一截断，各工具不再自行截断
- bash 工具的特殊 maxOut 逻辑迁移到统一服务

### D7: Invalid 工具 — 在 builtin.go 注册

**选择**：注册名为 `invalid` 的内置工具，接受 `tool` 和 `error` 参数，返回错误描述。Engine 在工具 schema 校验失败时路由到 invalid。

**替代方案**：直接返回错误字符串 → 模型无法从结构化反馈中学习。

### D8: Read 工具增强 — 添加 offset/limit

**选择**：在 read 工具 schema 中添加 `offset`（起始行号，1-based）和 `limit`（读取行数）可选参数。未提供时读取全文件。

### D9: 模块接入 — wire.go 重构

**选择**：在 `wire.go` 中依次接入：
1. **Snapshot**：创建 `snapshot.New()`，赋值给 `eng.Snapshot`
2. **LSP**：若配置了 LSP 服务器命令，创建 `lsp.NewClient()`，调用 `tool.RegisterLSP()`
3. **FileWatcher**：创建 `filewatcher.New()`，传入 Bus；在 write/edit/apply_patch 工具中注入 watcher 引用
4. **Revert + Snapshot**：修改 `Store.Revert()` 或在上层调用 `Snapshot.Restore()`

### D10: 配置增强 — JSONC + 新字段

**选择**：使用 `github.com/tailscale/hujson` 库支持 JSONC 解析（去除注释和尾逗号后转标准 JSON）。新增配置字段在 `x_opencode_go` 命名空间下。

**新增字段**：
- `model`：默认模型（`"provider/model"` 格式）
- `small_model`：小模型（用于 compaction/title/summary）
- `instructions`：指令数组（文件路径或 URL）
- `compaction`：`{auto: bool, reserved: int, prune: bool}`
- `lsp`：`{servers: [{language: string, command: string, args: []string}]}`

## Risks / Trade-offs

- **系统提示 base prompt 与 TS 不同步**：TS 有大量模型特定的 prompt 文本（BEAST/CODEX/GEMINI 等），Go 仅需 OpenAI 和 Anthropic 两套。→ 后续可按需添加。
- **Compaction token 计算依赖 Provider 返回**：Go SDK 返回的 Usage 中 token 计数可能不够精确。→ 使用保守的 reserved 值（20000）作为缓冲。
- **LSP 进程管理复杂度**：LSP 服务器可能挂起或崩溃。→ 实现超时和进程健康检查，失败时静默降级。
- **JSONC 引入新依赖**：hujson 是 Tailscale 维护的成熟库，风险低。
- **Agent 定义与 Mode 并存**：从 Mode 迁移到 Agent 需要更新 Engine 和 wire.go。→ Agent 包含 Mode 作为属性，保持向后兼容。
