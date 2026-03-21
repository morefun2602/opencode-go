# agent-modes Specification

## Purpose

定义 Agent 模式（build/plan/explore）、工具标签过滤与自定义模式配置。

## Requirements

### Requirement: 模式定义

系统 MUST 支持至少三种 Agent 模式：`build`（默认，允许全部工具）、`plan`（禁止写操作工具）、`explore`（仅允许只读操作）。每种模式 MUST 定义允许的工具标签集合。Mode MUST 作为 Agent 的一个属性存在，Agent MUST 成为 Engine 的运行模式单元。

#### Scenario: build 模式

- **WHEN** 当前 Agent 的 Mode 为 `build`
- **THEN** 所有已注册工具 MUST 对模型可见（除非 Agent ToolPermissions 另有限制）

#### Scenario: plan 模式

- **WHEN** 当前 Agent 的 Mode 为 `plan`
- **THEN** 工具列表 MUST 排除标签为 `write` 的工具（edit、write、apply_patch、bash）

#### Scenario: explore 模式

- **WHEN** 当前 Agent 的 Mode 为 `explore`
- **THEN** 工具列表 MUST 仅包含标签为 `read` 的工具（read、glob、grep、webfetch、websearch）

### Requirement: Agent 结构体定义

系统 MUST 定义 `Agent` 结构体（`internal/runtime/agent.go`），包含 Name、Prompt（可选的自定义系统提示）、Mode（build/plan/explore）、Hidden（是否对用户隐藏）、ToolPermissions（工具权限规则）字段。Agent MUST 替代现有 Mode 作为 Engine 的运行模式单元。

#### Scenario: Agent 包含所有必要字段

- **WHEN** 系统启动并初始化 Agent 列表
- **THEN** 每个 Agent MUST 包含 Name、Mode、Hidden 和 ToolPermissions 字段

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

系统 MUST 实现 `ToolFilter(agent Agent, allTools []ToolDef) []ToolDef` 函数，根据 Agent 的 ToolPermissions 规则（allow/deny 列表）过滤工具。该函数 MUST 替代现有的纯 Tags 过滤。

#### Scenario: deny 列表过滤

- **WHEN** Agent 的 ToolPermissions 包含 deny 列表 `["todowrite"]`
- **THEN** 过滤后的工具列表 MUST 不包含 todowrite

#### Scenario: 全部拒绝

- **WHEN** Agent 的 ToolPermissions 为 `deny: ["*"]`
- **THEN** 过滤后的工具列表 MUST 为空

### Requirement: 工具标签

每个内置工具 MUST 声明其标签集合（`read`、`write`、`execute`）。Engine 在收集工具定义时 MUST 根据当前模式过滤工具列表。

#### Scenario: 工具注册含标签

- **WHEN** 工具注册时
- **THEN** 工具定义 MUST 包含 `Tags []string` 字段

### Requirement: 模式切换

用户 MUST 可以在会话内通过 TUI 快捷键或 API 切换模式。模式切换 MUST 立即生效于下一次 turn。

#### Scenario: TUI 中切换模式

- **WHEN** 用户在 TUI 中按模式切换快捷键
- **THEN** 当前会话的模式 MUST 更新，后续 turn 的工具列表 MUST 反映新模式

### Requirement: 自定义模式

系统 MUST 支持通过 `x_opencode_go.agents` 配置自定义 Agent 模式，每个自定义模式可指定允许的工具名单、模型、温度等参数。

#### Scenario: 自定义模式加载

- **WHEN** 配置包含自定义 Agent 模式定义
- **THEN** 该模式 MUST 在模式列表中可用且按配置过滤工具
