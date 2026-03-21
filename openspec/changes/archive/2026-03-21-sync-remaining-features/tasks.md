## 1. 系统提示构建（system-prompt）

- [x] 1.1 新建 `internal/prompt/` 包，实现 `ModelPrompt(providerType string) string`：为 OpenAI 和 Anthropic 分别提供 base prompt 文本（嵌入为 Go 常量或 embed 文件）
- [x] 1.2 实现 `EnvironmentPrompt(workspaceRoot string) string`：注入工作目录路径、操作系统平台（runtime.GOOS）、当前日期、git 分支名和状态（通过 `git rev-parse --abbrev-ref HEAD` 和 `git status --porcelain`）
- [x] 1.3 实现 `InstructionPrompt(workspaceRoot string, configInstructions []string) string`：向上查找 AGENTS.md / CLAUDE.md / CONTEXT.md（从工作区根到文件系统根），加载 config.instructions 中的文件路径（相对工作区）和 URL（HTTP GET）
- [x] 1.4 实现 `SkillSummary(skills []skill.Skill) string`：仅输出技能名称和描述列表（不含完整正文）
- [x] 1.5 实现 `Build(opts BuildOpts) string`：按顺序组装 ModelPrompt → AgentPrompt → EnvironmentPrompt → InstructionPrompt → SkillSummary
- [x] 1.6 修改 `internal/runtime/engine.go`，将系统提示构建从 `skill.InjectPrompt(e.SystemPrompt, e.Skills)` 替换为 `prompt.Build()`
- [x] 1.7 编写 `internal/prompt/` 模块的单元测试

## 2. Provider 路由与模型选择（provider-routing）

- [x] 2.1 在 `internal/llm/` 新建 `routing.go`，定义 `ModelRef{ProviderID, ModelID string}` 和 `Router` 结构体
- [x] 2.2 实现 `ParseModel(s string) ModelRef`：解析 `"provider/model"` 格式，无 `/` 时 ProviderID 为空
- [x] 2.3 实现 `Router.DefaultModel() ModelRef`：config.model → 第一个可用 provider 的第一个模型
- [x] 2.4 实现 `Router.SmallModel() ModelRef`：config.small_model → 按 provider 类型自动选择（Anthropic: haiku 系列，OpenAI: gpt-4o-mini 系列）
- [x] 2.5 实现 `Router.Resolve(ref ModelRef) (Provider, string, error)`：按 ProviderID 精确匹配或搜索所有 provider
- [x] 2.6 修改 `internal/runtime/engine.go`，将 `LLM Provider` 字段替换为 `Router *llm.Router`，所有 LLM 调用通过 Router 获取 Provider
- [x] 2.7 修改 `internal/cli/wire.go`，构建 `llm.Router` 并注入 Engine
- [x] 2.8 编写 Provider 路由的单元测试

## 3. Agent 定义（agent-modes 增强）

- [x] 3.1 在 `internal/runtime/` 新建 `agent.go`，定义 `Agent` 结构体（Name, Prompt, Mode, Hidden, ToolPermissions）
- [x] 3.2 定义 7 个内置 Agent：build、plan、explore、general、compaction、title、summary，含各自的 prompt 和权限规则
- [x] 3.3 实现 `ToolFilter(agent Agent, allTools []ToolDef) []ToolDef`：根据 Agent 的 deny/allow 列表和 Mode Tags 过滤工具
- [x] 3.4 修改 `internal/runtime/engine.go`，将 `Mode` 字段替换为 `Agent`，`collectTools()` 使用 `ToolFilter` 替代纯 Tags 过滤
- [x] 3.5 创建 Agent prompt 文本文件：`internal/prompt/explore.txt`（强调 glob/grep/read）、`internal/prompt/compaction.txt`（总结模板）、`internal/prompt/title.txt`（≤50 字符标题）、`internal/prompt/summary.txt`（2-3 句摘要）
- [x] 3.6 编写 Agent 定义和 ToolFilter 的单元测试

## 4. Compaction 会话集成（react-loop 增强）

- [x] 4.1 在 `internal/tools/compaction.go` 中实现 `IsOverflow(usage llm.Usage, modelLimit, reserved int) bool`：基于 token 使用量判断
- [x] 4.2 实现 `Prune(msgs []llm.Message, keepRecentTokens int) []llm.Message`：从旧到新裁剪 tool_result content 为 `[pruned]`，跳过 skill 工具结果，保留最近约 40000 tokens
- [x] 4.3 重构 `Compactor.Process()`：使用 compaction agent（通过 Router.SmallModel()）生成摘要，替代通用 prompt
- [x] 4.4 在 `internal/runtime/engine.go` 的 ReAct 循环中集成完整 Compaction 流程：每轮结束后检查 IsOverflow → Prune → Process
- [x] 4.5 增加配置支持：读取 config.compaction.auto / reserved / prune 参数
- [x] 4.6 编写 Compaction 集成的单元测试

## 5. Provider 消息转换（provider-transform）

- [x] 5.1 在 `internal/llm/` 新建 `transform.go`，实现 `TransformMessages(msgs []Message, providerType string) []Message`
- [x] 5.2 实现 Anthropic 规则：过滤空 text content、toolCallId 仅保留 `[a-zA-Z0-9_-]` 字符
- [x] 5.3 在 `internal/llm/anthropic.go` 的 Chat/ChatStream 方法中调用 TransformMessages
- [x] 5.4 编写消息转换的单元测试

## 6. 统一截断服务（tool-truncation）

- [x] 6.1 新建 `internal/truncate/` 包，实现 `Truncate(output string, opts Options) Result`，支持 MaxLines(2000)/MaxBytes(50KB)/Direction(head/tail)
- [x] 6.2 修改 `internal/tools/registry.go` 的 `Run()` 方法，在工具执行返回后统一调用截断服务
- [x] 6.3 移除 `internal/tool/builtin.go` 中 grep/bash 的局部截断逻辑
- [x] 6.4 移除 `internal/tool/webfetch.go`、`internal/tool/ls.go` 中的局部截断逻辑
- [x] 6.5 编写截断服务的单元测试

## 7. Invalid 工具（builtin-tools 增强）

- [x] 7.1 在 `internal/tool/` 新建 `invalid.go`，实现 invalid 工具：接受 tool/error 参数，返回描述性错误
- [x] 7.2 在 `internal/tool/builtin.go` 中注册 invalid 工具（不加入活跃工具列表）
- [x] 7.3 修改 `internal/runtime/engine.go` 或 `internal/tool/router.go`，在工具调用参数解析失败时路由到 invalid 工具
- [x] 7.4 编写 invalid 工具的单元测试

## 8. Read 工具增强（builtin-tools 增强）

- [x] 8.1 修改 `internal/tool/builtin.go` 中 read 工具的 schema，添加 `offset`（integer，可选）和 `limit`（integer，可选）参数
- [x] 8.2 实现 offset/limit 逻辑：按行分割文件内容，支持 offset（1-based 起始行）和 limit（读取行数）
- [x] 8.3 实现单行截断：超过 2000 字符的行截断并附加 `... (line truncated)`
- [x] 8.4 编写 read 工具增强的单元测试

## 9. 配置增强（cli-and-config 增强）

- [x] 9.1 在 go.mod 中添加 `github.com/tailscale/hujson` 依赖
- [x] 9.2 修改 `internal/config/config.go` 的配置加载逻辑：使用 hujson.Standardize() 在 json.Unmarshal 前去除注释和尾逗号
- [x] 9.3 在 Config 结构体中添加新字段：`Model string`、`SmallModel string`、`Instructions []string`（增强）、`Compaction CompactionConfig{Auto bool, Reserved int, Prune bool}`、`LSP LSPConfig{Servers []LSPServer}`
- [x] 9.4 编写 JSONC 解析和新配置字段的单元测试

## 10. 模块接入 — Snapshot（snapshot-restore 增强）

- [x] 10.1 修改 `internal/cli/wire.go`：在工作区为 git 仓库时创建 `snapshot.New()` 实例并赋值给 `eng.Snapshot`
- [x] 10.2 修改会话 Revert 调用方（HTTP handler 或 CLI），在 Store.Revert() 后调用 `eng.Snapshot.Restore()` 恢复文件状态
- [x] 10.3 处理 Snapshot 不可用时的优雅降级：Revert 仅回退消息不报错
- [x] 10.4 编写 Snapshot 接入的集成测试

## 11. 模块接入 — LSP（lsp-integration 增强）

- [x] 11.1 修改 `internal/cli/wire.go`：当配置包含 LSP 服务器时创建 `lsp.NewClient()` 并调用 `tool.RegisterLSP()`
- [x] 11.2 实现 LSP 客户端生命周期管理：Engine 关闭时调用 Client.Close()
- [x] 11.3 处理 LSP 进程异常退出的优雅降级
- [x] 11.4 编写 LSP 接入的集成测试

## 12. 模块接入 — FileWatcher（file-monitoring 增强）

- [x] 12.1 修改 `internal/cli/wire.go`：创建 `filewatcher.New()` 实例并传入 Bus
- [x] 12.2 修改 `internal/tool/builtin.go` 中 write 工具的实现：写入成功后调用 `watcher.NotifyChange(path)`
- [x] 12.3 修改 `internal/tool/builtin.go` 中 edit 工具的实现：编辑成功后调用 `watcher.NotifyChange(path)`
- [x] 12.4 修改 `internal/tool/apply_patch.go`：每个被修改文件成功后调用 `watcher.NotifyChange(path)`
- [x] 12.5 修改工具注册函数签名以接受 `*filewatcher.Watcher` 参数（可为 nil）
- [x] 12.6 编写 FileWatcher 事件发布的单元测试

## 13. 集成验证

- [x] 13.1 运行 `go build ./...` 确保编译通过
- [x] 13.2 运行 `go test ./...` 确保全部单元测试通过
- [x] 13.3 运行 `go vet ./...` 检查代码质量
- [x] 13.4 手动测试系统提示构建：验证环境信息、InstructionPrompt、技能列表注入
- [x] 13.5 手动测试 Provider 路由：验证 defaultModel/smallModel 选择
- [x] 13.6 手动测试 read 工具 offset/limit
