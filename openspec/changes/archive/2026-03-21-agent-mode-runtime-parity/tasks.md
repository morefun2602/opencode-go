## 1. Permission 包（D1）

- [x] 1.1 新建 `internal/permission/permission.go`，定义 `Action`（allow/deny/ask）、`Rule`、`Ruleset` 类型
- [x] 1.2 实现 `matchPattern(pattern, target string) bool`：`*` 全匹配、`prefix*` 前缀匹配、精确匹配
- [x] 1.3 实现 `toolPermissionName(toolName string) string`：edit/write/apply_patch/multiedit → "edit"，其他返回原名
- [x] 1.4 实现 `Disabled(toolNames []string, ruleset Ruleset) map[string]bool`：findLast 匹配，deny 加入结果集
- [x] 1.5 实现 `Evaluate(permission, target string, ruleset Ruleset) Action`：findLast 匹配，无匹配默认 allow
- [x] 1.6 实现 `Merge(defaults, overrides Ruleset) Ruleset`：append(defaults, overrides...)
- [x] 1.7 编写 `internal/permission/permission_test.go`：覆盖全通配、前缀匹配、精确匹配、编辑工具统一映射、多规则覆盖、Merge、Evaluate

## 2. Agent 结构体增强（D2）

- [x] 2.1 修改 `internal/runtime/agent.go`：Agent 新增 Description、Steps、Model、Temperature（`*float64`）、Subagent（bool）、Permission（`permission.Ruleset`）字段
- [x] 2.2 移除 `ToolPermission` 结构体
- [x] 2.3 迁移内置 Agent 的 Permission：AgentBuild（nil）、AgentPlan（deny edit+execute, allow plan_enter/plan_exit）、AgentExplore（deny *, allow read/bash/skill/ls）、AgentGeneral（deny todowrite/todoread）、隐藏 Agent（deny *）
- [x] 2.4 为 explore 和 general 设置 `Subagent: true`
- [x] 2.5 更新 `ListAgents()` 返回结果包含 Description

## 3. ToolFilter 迁移（D7）

- [x] 3.1 修改 `ToolFilter`：Permission 非空时使用 `permission.Disabled` 过滤；为 nil 时 fallback 到 Mode Tags
- [x] 3.2 更新 `internal/runtime/agent_test.go`：用 Permission Ruleset 替代 ToolPermission 测试用例
- [x] 3.3 新增 Permission 过滤的测试：deny edit 排除所有编辑工具、deny * 排除所有、allow read 仅保留 read

## 4. AgentFile 配置增强（D10）

- [x] 4.1 修改 `internal/config/config.go`：AgentFile 新增 Steps、Prompt、Mode、Hidden、Subagent、Description 字段
- [x] 4.2 更新 `merge()` 中 Agents 合并逻辑

## 5. 自定义 Agent 注册（D3）

- [x] 5.1 修改 `internal/runtime/agent.go`：新增 `RegisterAgent(a Agent)` 函数，将 builtinAgents 改为可修改的 registry
- [x] 5.2 修改 `internal/cli/wire.go`：遍历 `cfg.Agents`，将每个 AgentFile 转为 Agent 并调用 RegisterAgent
- [x] 5.3 AgentFile.Tools 转为 allow-only Ruleset：先 deny *，再逐一 allow
- [x] 5.4 AgentFile.Mode 映射到 Mode 结构体（默认 ModeBuild）

## 6. AgentSwitch — session-level Agent 管理（D4）

- [x] 6.1 新建 `internal/runtime/agent_switch.go`：定义 AgentSwitch 结构体（sync.RWMutex + map[string]Agent）
- [x] 6.2 实现 Get/Set/Delete 方法
- [x] 6.3 Engine 新增 `AgentSwitch *AgentSwitch` 字段
- [x] 6.4 实现 `Engine.activeAgent(ctx, sessionID string) Agent`：先查 context override → AgentSwitch → e.Agent → AgentBuild
- [x] 6.5 在 CompleteTurn/CompleteTurnStream 结束时 defer `AgentSwitch.Delete(sessionID)`

## 7. Engine Agent 感知 — collectTools & buildSystemPrompt（D4, D5）

- [x] 7.1 修改 `collectTools()` 为 `collectToolsForAgent(agent Agent)`：使用 `activeAgent(ctx, sessionID)` 获取当前 Agent
- [x] 7.2 修改 CompleteTurn/CompleteTurnStream 的循环：每轮重新获取 activeAgent 和 tools
- [x] 7.3 修改 `buildSystemPrompt` 为 `buildSystemPromptForAgent`：接受 Agent 参数
- [x] 7.4 实现 `resolveProviderForAgent(agent Agent) (llm.Provider, string)`
- [x] 7.5 实现 `maxRounds(agent Agent) int`：Agent.Steps > 0 时使用 Steps
- [x] 7.6 在循环的 LLM 调用前使用 `resolveProviderForAgent` 选择 Provider
- [x] 7.7 在循环的 maxRounds 判断中使用 `maxRounds(activeAgent)`

## 8. plan_enter/plan_exit 联动（D8）

- [x] 8.1 修改 `internal/tool/plan.go`：`registerPlan` 新增 PlanSwitch 接口参数
- [x] 8.2 plan_enter 改为 `PlanSwitch.EnterPlan(sessionID)`（设置 AgentPlan）
- [x] 8.3 plan_exit 改为 `PlanSwitch.ExitPlan(sessionID)`（恢复默认）
- [x] 8.4 移除 `GlobalModeSwitch` 和 `ModeSwitch` 类型
- [x] 8.5 更新 `wire.go`：通过 planSwitchAdapter 将 AgentSwitch 传入 RegisterPlan

## 9. task 工具 subagent_type 实现（D6）

- [x] 9.1 修改 `internal/tool/task.go`：RegisterTask 新增 listSubagents/validateSubagent 回调
- [x] 9.2 实现 subagent_type 解析：有效且 CanUse=true → 使用该 Agent 名；为空 → 使用 general；无效/不可调用 → 返回错误
- [x] 9.3 通过 context value 传递选定 Agent 名给子 Engine
- [x] 9.4 Engine.CompleteTurn 入口通过 activeAgent 检查 context 中的 subagent name，优先使用
- [x] 9.5 更新 `wire.go`：将 listSubagents/validateSubagent 闭包传入 RegisterTask

## 10. Confirm 注入（D9）

- [x] 10.1 修改 `internal/cli/wire.go`：engine.Confirm 已作为字段可由调用方设置
- [x] 10.2 确保 HTTP API 模式下 `engine.Confirm` 使用 tools.Questions 异步机制
- [x] 10.3 Engine executeTool 中 agent-level permission ask：Confirm 非 nil 时调用；为 nil 时默认拒绝

## 11. 集成验证

- [x] 11.1 运行 `go build ./...` 确保编译通过
- [x] 11.2 运行 `go test ./internal/permission/...` 确保 Permission 包测试通过
- [x] 11.3 运行 `go test ./internal/runtime/...` 确保 runtime 测试通过
- [x] 11.4 运行 `go vet ./...` 检查代码质量
- [x] 11.5 验证 plan_enter/plan_exit 切换后工具列表变化
- [x] 11.6 验证 task 工具的 subagent_type 参数选择不同 Agent
- [x] 11.7 验证自定义 Agent 配置可通过 GetAgent 获取
