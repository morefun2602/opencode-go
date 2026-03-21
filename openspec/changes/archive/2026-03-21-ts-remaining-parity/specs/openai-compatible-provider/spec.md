# Capability: openai-compatible-provider

## ADDED Requirements

### Requirement: OpenAI 兼容提供商

系统 MUST 提供一个通用的 OpenAI-compatible Provider 实现，通过配置 `base_url` 和 `api_key` 即可接入任何兼容 OpenAI Chat Completions API 的服务（Azure OpenAI、Groq、DeepInfra、Together AI 等）。该提供商 MUST 复用现有 `openai-go` SDK，仅覆盖 BaseURL。

#### Scenario: 自定义 base_url 接入

- **WHEN** 配置中存在 `providers.groq` 条目且 `base_url` 指向 Groq API
- **THEN** 系统 MUST 使用该 base_url 初始化 OpenAI SDK 客户端并成功发起 Chat 请求

#### Scenario: 无 base_url 时回退

- **WHEN** 配置中某提供商未设置 `base_url` 且提供商名称不是 `openai` 或 `anthropic`
- **THEN** 系统 MUST 返回配置错误

### Requirement: Provider Registry

系统 MUST 维护一个 provider registry，支持按名称注册和查找 Provider 实例。`NewProvider` 工厂函数 MUST 支持 `openai`、`anthropic`、`openai-compatible`、`stub` 四种类型。对于未知的提供商名称且配置中包含 `base_url`，MUST 自动使用 openai-compatible 类型。

#### Scenario: 动态注册与查找

- **WHEN** 配置包含 `providers.groq` 条目（含 base_url 和 api_key）
- **THEN** registry MUST 创建 openai-compatible 类型的 Provider 并可通过名称 `groq` 查找

### Requirement: Model Listing

每个 Provider MUST 实现 `Models() []string` 方法，返回该提供商支持的模型列表。对 OpenAI 和 openai-compatible 类型，MUST 调用 `/v1/models` API 获取；对 Anthropic，MUST 返回硬编码的已知模型列表。

#### Scenario: 列出模型

- **WHEN** 调用某提供商的 `Models()` 方法
- **THEN** MUST 返回非空的模型标识符数组
