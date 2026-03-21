## ADDED Requirements

### Requirement: Agent 结构体增强字段

`Agent` 结构体 MUST 新增以下字段：
- `Description string` — 人类可读的 Agent 描述
- `Steps int` — 该 Agent 的最大 ReAct 循环轮数，为 0 时使用 Engine 默认值
- `Model string` — 该 Agent 专用模型（`"provider/model"` 格式），为空时使用 Engine 默认模型
- `Temperature *float64` — 该 Agent 的温度参数，nil 时使用默认值
- `Permission Ruleset` — 替代 ToolPermission，使用 Permission Ruleset 进行工具过滤

#### Scenario: Steps 限制轮数

- **WHEN** Agent 的 Steps 为 10 且 Engine 默认 MaxToolRounds 为 25
- **THEN** 使用该 Agent 的循环 MUST 最多执行 10 轮

#### Scenario: Model 覆盖默认

- **WHEN** Agent 的 Model 为 "openai/gpt-4o" 且 Engine 默认模型为 "anthropic/claude-sonnet-4-20250514"
- **THEN** 使用该 Agent 执行 LLM 调用时 MUST 使用 "openai/gpt-4o"

### Requirement: 自定义 Agent 注册

系统 MUST 在启动时将 `config.Agents`（`[]AgentFile`）中的每个条目转换为 `Agent` 并注册到全局 Agent 表中。自定义 Agent MUST 可通过 `GetAgent()` 获取。自定义 Agent 与内置 Agent 同名时，自定义 MUST 覆盖内置。

#### Scenario: 自定义 Agent 覆盖内置

- **WHEN** 配置定义了名为 "build" 的自定义 Agent
- **THEN** `GetAgent("build")` MUST 返回自定义版本

#### Scenario: 新自定义 Agent

- **WHEN** 配置定义了名为 "review" 的自定义 Agent
- **THEN** `GetAgent("review")` MUST 返回该 Agent
- **AND** `ListAgents()` MUST 包含 "review"

### Requirement: 运行时模式切换生效

`plan_enter`/`plan_exit` 工具的模式切换 MUST 实际影响 Engine 的工具收集。Engine MUST 在每轮循环开始时查询当前 session 的活跃 Agent，而非始终使用固定的 `e.Agent`。

#### Scenario: plan_enter 切换后工具过滤

- **WHEN** session 当前为 build 模式且 LLM 调用 plan_enter
- **THEN** 下一轮循环的 `collectTools()` MUST 使用 plan Agent 的权限过滤工具
- **AND** write 标签的工具 MUST 被排除

#### Scenario: plan_exit 恢复工具

- **WHEN** session 当前为 plan 模式且 LLM 调用 plan_exit 且用户确认
- **THEN** 下一轮循环的 `collectTools()` MUST 恢复 build Agent 的完整工具列表

### Requirement: explore Agent 工具限制

explore Agent 的工具列表 MUST 明确包含：read、glob、grep、webfetch、websearch、bash、skill、ls。MUST NOT 包含任何 write 标签工具。

#### Scenario: explore Agent 可用工具

- **WHEN** 使用 explore Agent 执行子任务
- **THEN** 工具列表 MUST 包含 read、glob、grep、bash
- **AND** MUST NOT 包含 edit、write、apply_patch

## MODIFIED Requirements

### Requirement: Agent 结构体定义

#### Scenario: Agent 包含所有必要字段

- **WHEN** 系统启动并初始化 Agent 列表
- **THEN** 每个 Agent MUST 包含 Name、Description、Mode、Hidden、Permission（Ruleset）、Steps、Model、Temperature 字段

### Requirement: Agent 级别工具过滤

#### Scenario: Permission Ruleset 过滤

- **WHEN** Agent 的 Permission Ruleset 包含 `{Permission: "edit", Pattern: "*", Action: "deny"}`
- **THEN** 所有 edit 类工具（edit、write、apply_patch、multiedit）MUST 被排除

#### Scenario: 通配符 pattern 过滤

- **WHEN** Agent 的 Permission Ruleset 包含 `{Permission: "*", Pattern: "*", Action: "deny"}`
- **THEN** 过滤后的工具列表 MUST 为空

### Requirement: 模式切换

#### Scenario: 模式切换通过 session Agent 覆盖

- **WHEN** 用户通过 plan_enter 切换模式
- **THEN** Engine MUST 在当前 session 中将活跃 Agent 替换为 plan Agent
- **AND** 替换 MUST 在下一轮 collectTools 时生效

### Requirement: 自定义模式

#### Scenario: AgentFile 完整配置

- **WHEN** 配置包含自定义 Agent `{name: "review", tools: ["read","grep"], model: "gpt-4o", temperature: 0.2}`
- **THEN** 该 Agent 的 Permission MUST 仅允许 read 和 grep
- **AND** Model MUST 为 "gpt-4o"
- **AND** Temperature MUST 为 0.2

## REMOVED Requirements

### Requirement: ToolPermission 结构体

`ToolPermission` 结构体（`Deny/Allow []string`）MUST 被移除，由 `Permission Ruleset` 替代。
