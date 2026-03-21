# Capability: session-management

## ADDED Requirements

### Requirement: 会话 Fork

系统 MUST 支持从指定会话的指定消息位置 fork 出新会话。新会话 MUST 复制源会话到指定消息为止的全部消息，并记录 `parent_id` 和 `parent_message_seq` 关联。

#### Scenario: Fork 成功

- **WHEN** 用户请求 fork 会话 A 在消息 seq=5 处
- **THEN** 系统 MUST 创建新会话 B，B 包含 A 的前 5 条消息的副本，B 的 `parent_id` 为 A 的 ID

#### Scenario: Fork 无效消息位置

- **WHEN** 指定的 seq 超出会话消息范围
- **THEN** 系统 MUST 返回错误

### Requirement: 会话 Revert

系统 MUST 支持将会话回滚到指定消息位置，删除该位置之后的所有消息。

#### Scenario: Revert 成功

- **WHEN** 用户请求 revert 会话到 seq=3
- **THEN** seq>3 的消息 MUST 被删除，会话可继续从 seq=3 后发起新对话

### Requirement: 会话元数据

会话 MUST 支持 `title`（字符串）和 `archived`（布尔值）字段。系统 MUST 提供更新这些字段的接口。

#### Scenario: 设置标题

- **WHEN** 用户或系统调用 setTitle
- **THEN** 会话的 title 字段 MUST 更新

#### Scenario: 归档会话

- **WHEN** 用户调用 setArchived(true)
- **THEN** 该会话 MUST 在默认列表中不可见，但可通过 `include_archived` 参数查询

### Requirement: 自动标题生成

系统 MUST 在会话首轮对话完成后自动使用 LLM 生成简短标题。

#### Scenario: 首轮后生成标题

- **WHEN** 会话的第一轮 turn 完成且 title 为空
- **THEN** 系统 MUST 异步调用 LLM 生成标题并更新会话

### Requirement: Usage 统计

系统 MUST 提供按会话统计 token 使用量与成本的能力，基于每条消息的 `cost_prompt_tokens` 和 `cost_completion_tokens` 列聚合。

#### Scenario: 查询 usage

- **WHEN** 客户端请求会话 usage
- **THEN** 系统 MUST 返回 `{prompt_tokens, completion_tokens, total_tokens}` 聚合数据
