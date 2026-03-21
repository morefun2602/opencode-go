# Proposal: Agent Mode & Runtime Parity

## Why

Go 版本的 Agent Mode 与 Agent Runtime 存在多处与 TypeScript 参考实现的关键差距，导致以下问题：

1. **运行时模式切换无效**：`plan_enter`/`plan_exit` 工具更新了 `GlobalModeSwitch`，但 Engine 的 `collectTools()` 从未读取该状态，始终使用固定的 `AgentBuild`，导致模式切换对工具过滤毫无效果。
2. **子 Agent 类型被忽略**：task 工具的 `subagent_type` 参数在 schema 中定义但实现中完全未使用，子任务始终使用主 Engine 的 Agent，无法按 explore/general 等类型运行。
3. **自定义 Agent 配置失效**：`config.Agents`（`AgentFile`）已解析但从未参与 Engine 的 Agent 选择，用户定义的自定义 Agent 无法生效。
4. **Permission 系统过于简单**：Go 仅有 `Deny/Allow []string` 列表，缺少 TS 的 pattern 通配符匹配（如 `"edit": "deny"`、`"internal-*": "deny"`）和 `ask` 交互行为。
5. **Agent 结构体不完整**：缺少 description、steps（最大迭代次数）、temperature、model 等字段，无法支持按 Agent 定制模型和行为。
6. **Engine 缺乏 Agent 感知**：系统提示、工具收集、模型选择均不感知当前 Agent，所有 session 行为一致。

## What Changes

### Core: Engine Agent 感知
- Engine 的 `collectTools()` 和 `buildSystemPrompt()` 读取当前活跃 Agent（包括运行时切换）
- 按 Agent 配置选择模型（agent.Model 覆盖默认）
- 按 Agent 的 steps 字段限制最大循环轮数

### Agent Mode: 运行时模式切换修复
- `plan_enter`/`plan_exit` 的模式切换与 Engine 联动，通过 session 级 Agent 覆盖生效
- collectTools 在每轮循环开始时重新查询当前 Agent

### SubAgent: task 工具 Agent 选择
- task 工具根据 `subagent_type` 参数选择对应 Agent（explore、general 等）
- 子 Engine 使用选定 Agent 的 Mode、Permission、Prompt

### Permission: 增强权限系统
- 引入 `Ruleset`（`[]Rule{Permission, Pattern, Action}`），支持 pattern 通配符
- 实现 `Permission.Disabled(tools, ruleset)` 用于工具过滤
- 支持 `ask` action 的交互式权限请求

### Config: 自定义 Agent 生效
- `config.Agents` 参与 Agent 注册，用户自定义 Agent 可通过 `GetAgent()` 获取
- Agent 配置支持 model、temperature、steps 等字段

## Capabilities

- **agent-modes** — Agent 定义增强、运行时模式切换修复、自定义 Agent 注册
- **agent-runtime** — Engine Agent 感知、按 Agent 选模型/限轮数、collectTools 动态化
- **permission** — Permission Ruleset 引入、pattern 匹配、ask 交互、disabled 工具过滤
- **task-tool** — subagent_type 实现、子 Engine Agent 选择、权限继承

## Impact

### Files to modify
- `internal/runtime/agent.go` — Agent 结构体增强、自定义 Agent 注册、Agent 查询增强
- `internal/runtime/mode.go` — 可能简化（Mode 合并入 Agent Permission）
- `internal/runtime/engine.go` — collectTools/buildSystemPrompt Agent 感知、按 Agent 选模型
- `internal/tool/plan.go` — ModeSwitch 与 Engine 联动
- `internal/tool/task.go` — subagent_type 实现
- `internal/cli/wire.go` — 自定义 Agent 注册、Agent 传递

### Files to create
- `internal/permission/permission.go` — Permission Ruleset、pattern 匹配、Disabled
- `internal/permission/permission_test.go` — Permission 单元测试

### Files to update (tests)
- `internal/runtime/agent_test.go` — Agent 增强测试
- `internal/tool/task_test.go` — subagent_type 测试
