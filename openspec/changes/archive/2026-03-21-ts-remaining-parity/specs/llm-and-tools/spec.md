# Capability: llm-and-tools (delta)

## MODIFIED Requirements

### Requirement: 提供商抽象

系统 MUST 将对语言模型提供商的出站调用隔离在 `Provider` 接口之后。接口为 `Chat(ctx, []Message, []ToolDef) (*Response, error)` 与 `ChatStream(ctx, []Message, []ToolDef, chunk func) (*Response, error)`，接受结构化消息数组与工具定义，返回包含 `Message`（含 Parts）、`Usage`、`Model`、`FinishReason` 的结构化响应。Provider 接口 MUST 新增 `Models() []string` 方法用于列出可用模型。

#### Scenario: 由配置选择提供商

- **WHEN** 配置指定某一受支持的提供商实现（openai / anthropic / openai-compatible / stub）
- **THEN** 运行时 MUST 将补全请求路由到该提供商，且除配置与提供商注册外无需改代码

#### Scenario: 接口支持工具定义

- **WHEN** 调用 `Chat` 并传入非空 `[]ToolDef`
- **THEN** Provider MUST 将工具定义传给模型 API，使模型能够返回 tool_calls

#### Scenario: 列出模型

- **WHEN** 调用 Provider 的 `Models()` 方法
- **THEN** MUST 返回该提供商支持的模型标识符列表

## ADDED Requirements

### Requirement: 工具模式过滤

Engine 在收集工具定义时 MUST 根据当前 Agent 模式的标签配置过滤工具列表。`collectTools` 方法 MUST 接受模式参数，仅返回标签匹配的工具。

#### Scenario: plan 模式过滤

- **WHEN** 当前模式为 `plan` 且某工具标签为 `["write"]`
- **THEN** 该工具 MUST NOT 出现在 Provider 收到的工具列表中
