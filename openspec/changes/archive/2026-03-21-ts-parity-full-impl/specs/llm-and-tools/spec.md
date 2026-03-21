# Capability: llm-and-tools (delta)

## MODIFIED Requirements

### Requirement: 提供商抽象

系统 MUST 将对语言模型提供商的出站调用隔离在 `Provider` 接口之后。接口从 `Complete(ctx, prompt string)` 变更为 `Chat(ctx, []Message, []ToolDef) (*Response, error)` 与 `ChatStream(ctx, []Message, []ToolDef, chunk func) (*Response, error)`，接受结构化消息数组与工具定义，返回包含 `Message`（含 Parts）、`Usage`、`Model`、`FinishReason` 的结构化响应。**BREAKING**

#### Scenario: 由配置选择提供商

- **WHEN** 配置指定某一受支持的提供商实现（openai / anthropic / stub）
- **THEN** 运行时 MUST 将补全请求路由到该提供商，且除配置与提供商注册外无需改代码

#### Scenario: 接口支持工具定义

- **WHEN** 调用 `Chat` 并传入非空 `[]ToolDef`
- **THEN** Provider MUST 将工具定义传给模型 API，使模型能够返回 tool_calls

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

### Requirement: 提供商出站 HTTP

对模型提供商的出站 HTTP 通过官方 Go SDK（`openai/openai-go`、`anthropics/anthropic-sdk-go`）发起，请求超时 MUST 来自配置或 context，并 MUST 遵守上下文取消。SDK 内部的 HTTP 客户端 MUST 受 context 控制。

#### Scenario: 提供商请求超时

- **WHEN** 提供商请求超过配置的超时时间
- **THEN** SDK MUST 取消该请求并向调用方返回归类为超时的错误

## ADDED Requirements

### Requirement: LLM 重试

Engine MUST 对 Provider 返回的可重试错误（RateLimit、Timeout）进行自动重试，最大重试次数由 `LLMMaxRetries` 配置，Auth 类错误 MUST NOT 重试。

#### Scenario: RateLimit 重试成功

- **WHEN** Provider 首次调用返回 RateLimit 错误
- **THEN** Engine MUST 重试至少一次，重试成功后正常返回结果
