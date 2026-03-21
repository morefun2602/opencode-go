# Capability: message-parts

## ADDED Requirements

### Requirement: 结构化 parts 存储

消息 MUST 以 JSON 序列化的 `parts` 数组（`[]Part`）存储结构化内容，取代纯文本 `body` 作为消息的主要数据载体。

#### Scenario: assistant 消息含 tool_call

- **WHEN** 助手消息包含文本与 tool_call 部件
- **THEN** `parts` MUST 包含 `type=text` 与 `type=tool_call` 的 Part 且 JSON 可被完整反序列化

### Requirement: 元数据列

消息表 MUST 包含 `model`（模型标识）、`cost_prompt_tokens`（输入 token 数）、`cost_completion_tokens`（输出 token 数）、`finish_reason` 列，由 Provider 响应填充。

#### Scenario: Usage 数据持久化

- **WHEN** Provider 返回 Usage 信息
- **THEN** 对应的 assistant 消息行 MUST 写入 `cost_prompt_tokens` 与 `cost_completion_tokens` 的非零值

### Requirement: tool_call_id 关联

`role=tool` 的消息 MUST 包含 `tool_call_id` 列，与产生该调用的 assistant 消息中对应的 tool_call ID 匹配。

#### Scenario: 结果关联

- **WHEN** 查询某条 tool_result 消息
- **THEN** 其 `tool_call_id` MUST 能在同一会话的 assistant 消息 parts 中找到匹配的 tool_call

### Requirement: schema v2 → v3 迁移

迁移脚本 MUST 将 schema v2 的纯文本 `body` 包装为 `[{"type":"text","text":"<body>"}]` 写入 `parts` 列，且 MUST NOT 删除原 `body` 列。

#### Scenario: 旧数据升级

- **WHEN** 打开 schema v2 的数据库
- **THEN** 迁移完成后所有既有消息 MUST 有有效的 `parts` JSON 且 `body` 保持不变
