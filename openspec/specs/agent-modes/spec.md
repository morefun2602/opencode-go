# agent-modes Specification

## Purpose

定义 Agent 模式（build/plan/explore）、工具标签过滤与自定义模式配置。

## Requirements

### Requirement: 模式定义

系统 MUST 支持至少三种 Agent 模式：`build`（默认，允许全部工具）、`plan`（禁止写操作工具）、`explore`（仅允许只读操作）。每种模式 MUST 定义允许的工具标签集合。Mode MUST 作为 Agent 的一个属性存在，Agent MUST 成为 Engine 的运行模式单元。

#### Scenario: build 模式

- **WHEN** 当前 Agent 的 Mode 为 `build`
- **THEN** 所有已注册工具 MUST 对模型可见（除非 Agent 的 Permission Ruleset 另有限制）

#### Scenario: plan 模式

- **WHEN** 当前 Agent 的 Mode 为 `plan`
- **THEN** 工具列表 MUST 排除标签为 `write` 的工具（edit、write、apply_patch、bash）

#### Scenario: explore 模式

- **WHEN** 当前 Agent 的 Mode 为 `explore`
- **THEN** 工具列表 MUST 明确包含：read、glob、grep、webfetch、websearch、bash、skill、ls
- **AND** MUST NOT 包含任何 write 标签工具

### Requirement: Agent 结构体定义

系统 MUST 定义 `Agent` 结构体（`internal/runtime/agent.go`），包含 Name、Description、Prompt（可选的自定义系统提示）、Mode（build/plan/explore）、Hidden（是否对用户隐藏）、Permission（Ruleset，用于工具过滤）、Steps、Model、Temperature 等字段。Agent MUST 替代现有 Mode 作为 Engine 的运行模式单元。

#### Scenario: Agent 包含所有必要字段

- **WHEN** 系统启动并初始化 Agent 列表
- **THEN** 每个 Agent MUST 包含 Name、Description、Mode、Hidden、Permission（Ruleset）、Steps、Model、Temperature 字段

### Requirement: general Agent

系统 MUST 定义名为 `general` 的 Agent，用作通用子 Agent 任务。该 Agent 的 Mode MUST 为 build，但 MUST 禁止 todoread 和 todowrite 工具。

#### Scenario: general Agent 工具过滤

- **WHEN** 使用 general Agent 执行子任务
- **THEN** 工具列表 MUST 排除 todoread 和 todowrite

### Requirement: compaction Agent

系统 MUST 定义名为 `compaction` 的隐藏 Agent，无工具（工具列表为空），有专用系统提示用于总结对话内容。

#### Scenario: compaction Agent 无工具

- **WHEN** 使用 compaction Agent
- **THEN** 工具列表 MUST 为空

#### Scenario: compaction Agent 有专用 prompt

- **WHEN** 使用 compaction Agent 生成摘要
- **THEN** 系统提示 MUST 包含对话总结的指令模板（Goal/Instructions/Discoveries/Accomplished/Relevant files）

### Requirement: title Agent

系统 MUST 定义名为 `title` 的隐藏 Agent，无工具，有专用系统提示用于生成 ≤50 字符的会话标题。

#### Scenario: title Agent 生成标题

- **WHEN** 使用 title Agent 处理会话消息
- **THEN** 系统提示 MUST 指示模型生成不超过 50 个字符的简短标题

### Requirement: summary Agent

系统 MUST 定义名为 `summary` 的隐藏 Agent，无工具，有专用系统提示用于生成 2-3 句话的会话摘要。

#### Scenario: summary Agent 生成摘要

- **WHEN** 使用 summary Agent 处理会话消息
- **THEN** 系统提示 MUST 指示模型生成类似 PR 描述的 2-3 句话摘要

### Requirement: Agent 级别工具过滤

系统 MUST 实现 `ToolFilter(agent Agent, allTools []ToolDef) []ToolDef` 函数，根据 Agent 的 Permission Ruleset 过滤工具。该函数 MUST 替代现有的纯 Tags 过滤。

#### Scenario: Permission Ruleset 过滤

- **WHEN** Agent 的 Permission Ruleset 包含 `{Permission: "edit", Pattern: "*", Action: "deny"}`
- **THEN** 所有 edit 类工具（edit、write、apply_patch、multiedit）MUST 被排除

#### Scenario: 通配符 pattern 过滤

- **WHEN** Agent 的 Permission Ruleset 包含 `{Permission: "*", Pattern: "*", Action: "deny"}`
- **THEN** 过滤后的工具列表 MUST 为空

### Requirement: 工具标签

每个内置工具 MUST 声明其标签集合（`read`、`write`、`execute`）。Engine 在收集工具定义时 MUST 根据当前模式过滤工具列表。

#### Scenario: 工具注册含标签

- **WHEN** 工具注册时
- **THEN** 工具定义 MUST 包含 `Tags []string` 字段

### Requirement: 模式切换

用户 MUST 可以在会话内通过 TUI 快捷键或 API 切换模式；`plan_enter`/`plan_exit` 触发的模式切换 MUST 通过在当前 session 中将活跃 Agent 替换为对应模式的 Agent 来实现。模式切换 MUST 立即生效于下一次 turn。

#### Scenario: TUI 中切换模式

- **WHEN** 用户在 TUI 中按模式切换快捷键
- **THEN** 当前会话的模式 MUST 更新，后续 turn 的工具列表 MUST 反映新模式

#### Scenario: 模式切换通过 session Agent 覆盖

- **WHEN** 用户通过 plan_enter 切换模式
- **THEN** Engine MUST 在当前 session 中将活跃 Agent 替换为 plan Agent
- **AND** 替换 MUST 在下一轮 collectTools 时生效

### Requirement: 自定义模式

系统 MUST 支持通过 `agents` 配置自定义 Agent 模式，每个自定义模式可指定允许的工具名单（映射为 Permission Ruleset）、模型、温度等参数。

#### Scenario: AgentFile 完整配置

- **WHEN** 配置包含自定义 Agent `{name: "review", tools: ["read","grep"], model: "gpt-4o", temperature: 0.2}`
- **THEN** 该 Agent 的 Permission MUST 仅允许 read 和 grep
- **AND** Model MUST 为 "gpt-4o"
- **AND** Temperature MUST 为 0.2

### Requirement: Agent 结构体增强字段

`Agent` 结构体 MUST 新增以下字段：
- `Description string` — 人类可读的 Agent 描述
- `Steps int` — 该 Agent 的最大 ReAct 循环轮数，为 0 时使用 Engine 默认值
- `Model string` — 该 Agent 专用模型（`"provider/model"` 格式），为空时使用 Engine 默认模型
- `Temperature *float64` — 该 Agent 的温度参数，nil 时使用默认值
- `Permission Ruleset` — 使用 Permission Ruleset 进行工具过滤（替代已移除的 `ToolPermission`）

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
