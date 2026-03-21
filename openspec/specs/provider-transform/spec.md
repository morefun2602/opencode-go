# provider-transform Specification

## Purpose

定义 Per-Provider 消息规范化处理，包括 Anthropic 空 content 过滤、toolCallId 规范化、Provider 内部调用以及空消息列表安全处理。

## Requirements

### Requirement: Per-Provider 消息规范化

系统 MUST 提供 `internal/llm/transform.go` 模块，在发送消息给 LLM 前对消息列表进行 per-provider 规范化处理。

#### Scenario: Anthropic 空 content 过滤

- **WHEN** 消息列表中某条消息的 Content 为空字符串且目标 provider 为 Anthropic
- **THEN** 系统 MUST 过滤或替换该消息，MUST NOT 发送空 content 给 Anthropic API

#### Scenario: Anthropic toolCallId 规范化

- **WHEN** tool_call 的 ID 包含 Anthropic 不支持的字符
- **THEN** 系统 MUST 将 ID 规范化为仅含 `[a-zA-Z0-9_-]` 的字符串

#### Scenario: OpenAI 无需特殊处理

- **WHEN** 目标 provider 为 OpenAI
- **THEN** 系统 MUST 原样传递消息（SDK 已处理格式问题）

### Requirement: Provider 内部调用

消息转换 MUST 在 Provider.Chat() 和 Provider.ChatStream() 内部调用，对 Engine 和其他上层模块透明。

#### Scenario: 透明转换

- **WHEN** Engine 调用 Provider.Chat() 传入原始消息
- **THEN** Provider MUST 在内部调用 TransformMessages 后再发送给 API

### Requirement: 空消息列表安全

消息转换 MUST 能安全处理空消息列表和 nil 消息，MUST NOT panic。

#### Scenario: 空列表

- **WHEN** 传入空消息列表
- **THEN** MUST 返回空列表，MUST NOT 报错
