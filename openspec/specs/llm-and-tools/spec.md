# llm-and-tools Specification

## Purpose

定义 LLM 提供商抽象、工具调用契约与错误分类。

## Requirements

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

### Requirement: 流式补全

当上游 API 支持流式时，系统 MUST 消费流式分片并增量转发给会话消费者（CLI 或 HTTP），且 MUST NOT 在内存中缓冲完整补全。流式模式下 Provider MUST 通过回调函数增量传递包含部分文本或 tool_calls delta 的 `Response`。

#### Scenario: 完成前可见部分输出

- **WHEN** 某次补全启用流式（通过 `ChatStream`）
- **THEN** 消费者 MUST 在流成功结束之前通过 chunk 回调收到增量助手内容

### Requirement: 工具调用契约

系统 MUST 以显式名称与经 schema 校验的参数调用已注册工具。校验失败时系统 MUST NOT 执行该工具，并 MUST 向模型循环返回结构化错误（作为 `tool_result`），符合 agent-runtime ReAct 循环错误处理路径。

#### Scenario: 非法工具参数被拒绝

- **WHEN** 模型以未通过校验的参数请求工具
- **THEN** 系统 MUST NOT 执行该工具的副作用，并向编排层暴露校验错误作为 tool_result 消息

### Requirement: 工具与 MCP 失败语义

若工具或基于 MCP 的动作失败（网络错误、非零退出、超时），系统 MUST 在日志中记录失败并带关联 ID，MUST 将结构化失败描述回传给智能体循环以便模型按策略继续或停止。系统 MUST NOT 静默忽略工具失败。

#### Scenario: 记录工具失败

- **WHEN** 工具调用在已被接受后失败
- **THEN** 日志 MUST 包含含工具名与会话 ID 的错误级别条目，且智能体循环 MUST 收到与成功相区分的失败结果

### Requirement: 提供商出站 HTTP

对模型提供商的出站 HTTP 通过官方 Go SDK（`openai/openai-go`、`anthropics/anthropic-sdk-go`）发起，请求超时 MUST 来自配置或 context，并 MUST 遵守上下文取消。SDK 内部的 HTTP 客户端 MUST 受 context 控制。

#### Scenario: 提供商请求超时

- **WHEN** 提供商请求超过配置的超时时间
- **THEN** SDK MUST 取消该请求并向调用方返回归类为超时的错误

### Requirement: 多提供商注册表

系统 MUST 维护可扩展的 LLM 提供商注册表，且 MUST 允许通过配置选择提供商；注册表 MUST 与 `mcp-integration` 及内置工具路由在依赖上保持单向。

#### Scenario: 切换提供商

- **WHEN** 配置更改提供商标识并重启
- **THEN** 补全请求 MUST 路由到新提供商或 MUST 返回明确错误

### Requirement: 工具来源统一路由

系统 MUST 将内置工具与 MCP 工具统一纳入同一调用接口，使模型产生的 tool_calls 能解析到唯一实现；解析失败 MUST 返回结构化错误。

#### Scenario: 未知工具名

- **WHEN** 模型请求未注册工具名
- **THEN** 系统 MUST NOT 执行并 MUST 返回错误给编排层

### Requirement: 失败分类与可重试提示

系统 MUST 将提供商与工具错误分类（例如超时、429、认证失败）；对可重试错误，响应或日志 MUST 携带可重试提示。

#### Scenario: 超时可识别

- **WHEN** 提供商请求超时
- **THEN** 错误类型 MUST 与超时分类一致且 MUST 可被上层识别

### Requirement: LLM 重试

Engine MUST 对 Provider 返回的可重试错误（RateLimit、Timeout）进行自动重试，最大重试次数由 `LLMMaxRetries` 配置，Auth 类错误 MUST NOT 重试。

#### Scenario: RateLimit 重试成功

- **WHEN** Provider 首次调用返回 RateLimit 错误
- **THEN** Engine MUST 重试至少一次，重试成功后正常返回结果

### Requirement: 工具模式过滤

Engine 在收集工具定义时 MUST 根据当前 Agent 模式的标签配置过滤工具列表。`collectTools` 方法 MUST 接受模式参数，仅返回标签匹配的工具。

#### Scenario: plan 模式过滤

- **WHEN** 当前模式为 `plan` 且某工具标签为 `["write"]`
- **THEN** 该工具 MUST NOT 出现在 Provider 收到的工具列表中
