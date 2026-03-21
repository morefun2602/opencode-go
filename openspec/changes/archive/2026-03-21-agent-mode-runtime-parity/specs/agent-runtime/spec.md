## ADDED Requirements

### Requirement: Engine Agent 感知 — collectTools

Engine 的 `collectTools()` MUST 在每轮循环开始时查询当前 session 的活跃 Agent（通过 `ModeSwitch` 或 session-level override），而非始终使用固定的 `e.Agent`。活跃 Agent 变化时，工具列表 MUST 立即反映。

#### Scenario: session 级 Agent 覆盖

- **WHEN** session 通过 `plan_enter` 切换到 plan Agent
- **THEN** `collectTools()` 在该 session 的后续轮次 MUST 使用 plan Agent 的 Permission 过滤工具

#### Scenario: 无覆盖时使用默认

- **WHEN** session 无 Agent 覆盖
- **THEN** `collectTools()` MUST 使用 `e.Agent`（通常为 AgentBuild）

### Requirement: Engine Agent 感知 — 模型选择

Engine MUST 在执行 LLM 调用前检查当前活跃 Agent 的 `Model` 字段。若非空，MUST 通过 `Router` 获取对应 Provider 和模型 ID 替代默认 LLM。

#### Scenario: Agent 指定模型

- **WHEN** 活跃 Agent 的 Model 为 "openai/gpt-4o"
- **THEN** Engine MUST 使用 OpenAI Provider 的 gpt-4o 模型执行 LLM 调用

#### Scenario: Agent 未指定模型

- **WHEN** 活跃 Agent 的 Model 为空
- **THEN** Engine MUST 使用 `e.LLM` 或 `Router.DefaultModel()` 执行 LLM 调用

### Requirement: Engine Agent 感知 — 轮数限制

Engine MUST 在确定 `maxRounds` 时优先使用活跃 Agent 的 `Steps` 字段（若 > 0），否则使用 `e.MaxToolRounds`。

#### Scenario: Agent Steps 覆盖

- **WHEN** 活跃 Agent 的 Steps 为 10 且 Engine MaxToolRounds 为 25
- **THEN** 循环最多执行 10 轮

### Requirement: Engine Agent 感知 — buildSystemPrompt

`buildSystemPrompt()` MUST 使用活跃 Agent（而非固定 `e.Agent`）的 Prompt 字段。当 session 切换到不同 Agent 时，系统提示 MUST 反映新 Agent 的 Prompt。

#### Scenario: plan Agent 有 Prompt

- **WHEN** 活跃 Agent 为 plan 且 plan Agent 定义了自定义 Prompt
- **THEN** buildSystemPrompt MUST 使用 plan Agent 的 Prompt 替代 provider base prompt

### Requirement: session 级 Agent 管理

Engine MUST 维护 session → Agent 的映射（与现有 `ModeSwitch` 合并或替代）。提供 `GetSessionAgent(sessionID) Agent` 和 `SetSessionAgent(sessionID, Agent)` 方法。session 结束时 MUST 清理映射。

#### Scenario: session 结束清理

- **WHEN** CompleteTurn 执行完毕
- **THEN** session Agent 映射 MUST NOT 泄漏（defer 清理）

### Requirement: Confirm 函数注入

Engine 的 `Confirm` 字段 MUST 在 `wireEngine` 中被正确设置。当 Policy 要求 `ask` 且 `Confirm` 为 nil 时，MUST 默认拒绝。

#### Scenario: Confirm 在 REPL 中

- **WHEN** REPL 模式运行且工具权限为 ask
- **THEN** Engine.Confirm MUST 调用终端交互等待用户输入 y/n

#### Scenario: Confirm 在 HTTP 中

- **WHEN** HTTP API 模式运行
- **THEN** Engine.Confirm MUST 通过 Permission/Question 机制异步等待客户端响应

## MODIFIED Requirements

### Requirement: 会话生命周期

#### Scenario: CompleteTurn 使用活跃 Agent

- **WHEN** 调用 CompleteTurn 时 session 有 Agent 覆盖
- **THEN** Engine MUST 使用该 Agent 构建系统提示、收集工具、选择模型
- **AND** MUST NOT 使用固定的 `e.Agent`
