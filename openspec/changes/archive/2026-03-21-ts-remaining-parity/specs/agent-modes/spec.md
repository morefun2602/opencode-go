# Capability: agent-modes

## ADDED Requirements

### Requirement: 模式定义

系统 MUST 支持至少三种 Agent 模式：`build`（默认，允许全部工具）、`plan`（禁止写操作工具）、`explore`（仅允许只读操作）。每种模式 MUST 定义允许的工具标签集合。

#### Scenario: build 模式

- **WHEN** 当前模式为 `build`
- **THEN** 所有已注册工具 MUST 对模型可见

#### Scenario: plan 模式

- **WHEN** 当前模式为 `plan`
- **THEN** 工具列表 MUST 排除标签为 `write` 的工具（edit、write、apply_patch、bash）

#### Scenario: explore 模式

- **WHEN** 当前模式为 `explore`
- **THEN** 工具列表 MUST 仅包含标签为 `read` 的工具（read、glob、grep、webfetch、websearch）

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

系统 MUST 支持通过 `agents` 配置自定义 Agent 模式，每个自定义模式可指定允许的工具名单、模型、温度等参数。

#### Scenario: 自定义模式加载

- **WHEN** 配置包含自定义 Agent 模式定义
- **THEN** 该模式 MUST 在模式列表中可用且按配置过滤工具
