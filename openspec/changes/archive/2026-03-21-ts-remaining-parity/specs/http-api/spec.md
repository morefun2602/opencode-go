# Capability: http-api (delta)

## ADDED Requirements

### Requirement: 提供商与模型发现端点

系统 MUST 提供 `GET /v1/providers` 端点返回已注册提供商列表，以及 `GET /v1/providers/{id}/models` 端点返回指定提供商的可用模型列表。

#### Scenario: 列出提供商

- **WHEN** 客户端 GET `/v1/providers`
- **THEN** 响应 MUST 为 JSON 数组，每项包含提供商 `id` 和 `name`

#### Scenario: 列出模型

- **WHEN** 客户端 GET `/v1/providers/openai/models`
- **THEN** 响应 MUST 为 JSON 数组，每项包含模型标识符

### Requirement: 会话 Fork 端点

系统 MUST 提供 `POST /v1/sessions/{id}/fork` 端点，接受 `{message_seq}` 载荷，返回新创建的 fork 会话。

#### Scenario: Fork 成功

- **WHEN** 客户端 POST fork 请求且 message_seq 有效
- **THEN** 响应 MUST 为 201 且包含新会话 ID

### Requirement: 会话 Revert 端点

系统 MUST 提供 `POST /v1/sessions/{id}/revert` 端点，接受 `{message_seq}` 载荷，删除 seq 之后的消息。

#### Scenario: Revert 成功

- **WHEN** 客户端 POST revert 请求且 message_seq 有效
- **THEN** 响应 MUST 为 200

### Requirement: 会话元数据更新端点

系统 MUST 提供 `PATCH /v1/sessions/{id}` 端点，接受 `{title, archived}` 载荷更新会话元数据。

#### Scenario: 更新标题

- **WHEN** 客户端 PATCH `{"title": "新标题"}`
- **THEN** 会话标题 MUST 更新，响应 MUST 为 200

### Requirement: Usage 统计端点

系统 MUST 提供 `GET /v1/sessions/{id}/usage` 端点，返回该会话的 token 使用量聚合。

#### Scenario: 查询 usage

- **WHEN** 客户端 GET usage
- **THEN** 响应 MUST 包含 `prompt_tokens`、`completion_tokens`、`total_tokens` 字段

### Requirement: 配置读取端点

系统 MUST 提供 `GET /v1/config` 端点，返回当前运行时的非敏感配置信息（排除 API key）。

#### Scenario: 读取配置

- **WHEN** 客户端 GET `/v1/config`
- **THEN** 响应 MUST 为 JSON 且 MUST NOT 包含任何 API key 值
