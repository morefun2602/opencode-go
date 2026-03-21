# session-management Specification

## Purpose

定义会话的 Fork、Revert、元数据管理、自动标题生成与 Usage 统计能力。

## Requirements

### Requirement: 会话 Fork

系统 MUST 支持从指定会话的指定消息位置 fork 出新会话。新会话 MUST 复制源会话到指定消息为止的全部消息，并记录 `parent_id` 和 `parent_message_seq` 关联。

#### Scenario: Fork 成功

- **WHEN** 用户请求 fork 会话 A 在消息 seq=5 处
- **THEN** 系统 MUST 创建新会话 B，B 包含 A 的前 5 条消息的副本，B 的 `parent_id` 为 A 的 ID

#### Scenario: Fork 无效消息位置

- **WHEN** 指定的 seq 超出会话消息范围
- **THEN** 系统 MUST 返回错误

### Requirement: 会话 Revert

系统 MUST 支持将会话回滚到指定消息位置，删除该位置之后的所有消息。当 Snapshot 模块可用时，Revert MUST 同步恢复工作区文件状态到对应快照。系统 MUST 支持 unrevert 操作（撤销 revert）。

#### Scenario: Revert 成功

- **WHEN** 用户请求 revert 会话到 seq=3
- **THEN** seq>3 的消息 MUST 被删除，会话可继续从 seq=3 后发起新对话

#### Scenario: Revert 恢复文件

- **WHEN** 用户 revert 会话到 seq=3 且该位置有关联快照
- **THEN** 工作区文件 MUST 恢复到 seq=3 时的状态

#### Scenario: Unrevert

- **WHEN** 用户在 revert 后立即请求 unrevert
- **THEN** 系统 MUST 恢复被删除的消息和文件状态

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

### Requirement: 上下文压缩

系统 MUST 提供上下文压缩模块，当消息历史的 token 总量超过模型上下文窗口的阈值（默认 80%）时，MUST 通过 LLM 对早期消息生成摘要，用摘要替代原始消息，并保留最近 N 条消息（默认 5 条）不被压缩。

#### Scenario: Token 溢出触发压缩

- **WHEN** 消息历史 token 总量超过模型上下文窗口的 80%
- **THEN** 系统 MUST 调用 LLM 生成历史摘要，替代早期消息

#### Scenario: 压缩保留近期消息

- **WHEN** 压缩执行时
- **THEN** 最近 5 条消息 MUST 保留原样，仅早期消息被摘要替代

#### Scenario: 压缩结果持久化

- **WHEN** 压缩完成
- **THEN** 压缩后的消息序列 MUST 写入持久化层

### Requirement: 会话摘要生成

系统 MUST 在每个 ReAct step 完成后生成增量会话摘要，基于 Snapshot 的文件 diff 和工具调用结果。摘要 MUST 存储在会话元数据中。

#### Scenario: Step 完成后生成摘要

- **WHEN** 一个 ReAct step 完成且包含文件变更
- **THEN** 系统 MUST 生成该 step 的摘要并追加到会话摘要

#### Scenario: 无文件变更的 Step

- **WHEN** 一个 ReAct step 完成但无文件变更
- **THEN** 系统 MUST 仅记录工具调用信息到摘要

### Requirement: 增强重试逻辑

系统 MUST 提供增强的重试模块，支持：（1）retry-after 头解析；（2）指数退避策略；（3）按错误类型分类重试（RateLimit、Timeout、ServerError 可重试，Auth、InvalidRequest 不重试）。

#### Scenario: RateLimit 带 retry-after

- **WHEN** LLM 返回 429 错误且包含 retry-after: 5 头
- **THEN** 系统 MUST 等待 5 秒后重试

#### Scenario: 指数退避

- **WHEN** 连续多次重试
- **THEN** 等待时间 MUST 按指数增长（1s, 2s, 4s, 8s...）直到达到上限

#### Scenario: Auth 错误不重试

- **WHEN** LLM 返回 401/403 错误
- **THEN** 系统 MUST 立即返回错误，MUST NOT 重试
