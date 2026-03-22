# real-providers Specification

## Purpose

TBD

## Requirements

### Requirement: OpenAI 提供商

系统 MUST 提供基于 `openai/openai-go` 官方 SDK 的 OpenAI 提供商实现，支持 `Chat` 与 `ChatStream` 方法，将 `[]Message` + `[]ToolDef` 映射为 OpenAI chat completions 请求。

#### Scenario: 非流式调用成功

- **WHEN** 配置了有效的 OpenAI API key 且调用 `Chat`
- **THEN** 系统 MUST 返回包含 `Message`、`Usage`、`Model`、`FinishReason` 的 `Response`

#### Scenario: 流式调用含 tool_calls

- **WHEN** 调用 `ChatStream` 且模型返回 tool_calls
- **THEN** 最终返回的 `Response` MUST 包含完整拼接的 tool_calls（由 SDK 内部 accumulator 处理）

### Requirement: Anthropic 提供商

系统 MUST 提供基于 `anthropics/anthropic-sdk-go` 官方 SDK 的 Anthropic 提供商实现，将 Anthropic 特有的 `tool_use` / `tool_result` content block 映射为统一的 `Part` 结构。

#### Scenario: tool_use 映射

- **WHEN** Anthropic 模型返回 `tool_use` content block
- **THEN** 系统 MUST 将其映射为 `Part{Type: "tool_call", ...}` 且 `FinishReason` MUST 为 `"tool_calls"`

### Requirement: API key 配置优先级

两个提供商 MUST 支持以下 API key 来源优先级：配置文件 `providers.<name>.api_key` > 环境变量（`OPENAI_API_KEY` / `ANTHROPIC_API_KEY`）。

#### Scenario: 环境变量生效

- **WHEN** 配置文件未设置 API key 但环境变量已设置
- **THEN** 提供商 MUST 使用环境变量中的 key

### Requirement: base URL 覆盖

OpenAI 提供商 MUST 支持通过 `providers.openai.base_url` 配置覆盖默认 API 端点，以兼容 Azure OpenAI 或私有部署。

#### Scenario: 自定义 base URL

- **WHEN** 配置了 `base_url` 为私有端点
- **THEN** 所有 API 请求 MUST 发往该端点而非 `api.openai.com`
