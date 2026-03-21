## ADDED Requirements

### Requirement: 流式路径重试

Engine 的 `CompleteTurnStream` MUST 对 `ChatStream` 调用实现 Timeout/RateLimit 重试，与非流式路径 `callWithRetry` 行为对齐。重试 MUST 使用指数退避策略。

#### Scenario: 流式调用遇到 RateLimit

- **WHEN** `ChatStream` 返回 RateLimit 错误
- **THEN** Engine MUST 等待退避时间后重试，最多重试 `LLMMaxRetries` 次

#### Scenario: 流式调用遇到 Timeout

- **WHEN** `ChatStream` 返回 Timeout 错误
- **THEN** Engine MUST 重试，行为与非流式路径一致

### Requirement: 指数退避与 retry-after

Engine 的重试策略 MUST 使用指数退避（base 1s, factor 2, 上限 30s）。当 LLM 错误包含 `retry-after` 信息时，MUST 使用该值替代计算值。

#### Scenario: 指数退避

- **WHEN** 第 1 次重试
- **THEN** 等待约 1 秒

- **WHEN** 第 3 次重试
- **THEN** 等待约 4 秒（不超过 30 秒上限）

#### Scenario: retry-after 头

- **WHEN** LLM 返回携带 retry-after=5s 的 RateLimit 错误
- **THEN** Engine MUST 等待 5 秒后重试

### Requirement: Abort/Cancel 传播

Engine MUST 支持通过 `CancelSession(sessionID)` 取消正在运行的会话。取消 MUST 通过 `context.Context` 传播到 LLM 调用和工具执行。取消后 MUST 发布 `session.abort` Bus 事件。

#### Scenario: 用户取消会话

- **WHEN** 外部调用 `CancelSession` 且循环正在执行
- **THEN** 当前 LLM 调用 MUST 被 context 取消
- **AND** 循环 MUST 在当前轮结束后终止
- **AND** 已产生的消息 MUST 仍被持久化

#### Scenario: 取消时循环检查

- **WHEN** 循环每轮开始时检测到 `ctx.Err() != nil`
- **THEN** 循环 MUST 立即终止

### Requirement: filterCompacted 消息过滤

`loadHistory` MUST 在加载消息后过滤掉 compaction 点之前的消息。当消息内容包含 `[Conversation Summary]` 标记时，该消息及其后的消息为有效历史，之前的消息 MUST 被丢弃。

#### Scenario: 压缩后加载历史

- **WHEN** 历史中有 50 条消息，其中第 30 条为 compaction summary
- **THEN** `loadHistory` MUST 仅返回第 30-50 条消息

#### Scenario: 无 compaction 历史

- **WHEN** 历史中无 compaction 标记
- **THEN** `loadHistory` MUST 返回全部消息

### Requirement: maybeCompact 实际执行

`maybeCompact` MUST 在消息数超过阈值（`CompactionTurns * 2`）时实际执行 Compaction，而非仅记录日志。执行 MUST 异步进行以不阻塞主流程返回。

#### Scenario: 超过阈值触发压缩

- **WHEN** 会话消息数超过 `CompactionTurns * 2`
- **THEN** Engine MUST 异步调用 Compaction.Process 执行压缩

#### Scenario: 阈值为零时跳过

- **WHEN** `CompactionTurns <= 0`
- **THEN** `maybeCompact` MUST 不执行任何操作

### Requirement: MAX_STEPS 警告注入

当循环到达最后一轮（`round == maxRounds - 1`）时，Engine MUST 向消息列表追加警告消息，告知模型即将达到步数上限。最后一轮的 LLM 调用 MUST 使用空工具列表。

#### Scenario: 最后一轮注入警告

- **WHEN** 循环到达第 25 轮（MaxToolRounds=25）
- **THEN** Engine MUST 在消息列表末尾追加 MAX_STEPS 提示
- **AND** LLM 调用 MUST 传入空工具列表

#### Scenario: 非最后轮不受影响

- **WHEN** 循环在前 24 轮
- **THEN** Engine MUST 正常传入完整工具列表

### Requirement: Noop 工具注入

当 `collectTools()` 返回空列表但消息历史中存在 `tool_call` Part 时，Engine MUST 注入 `_noop` 占位工具。`_noop` 工具执行 MUST 返回 "noop" 字符串。

#### Scenario: 历史含 tool_calls 但当前无工具

- **WHEN** Agent 为 compaction（工具全部 deny）且历史消息含 tool_call
- **THEN** 工具列表 MUST 包含 `_noop` 工具

#### Scenario: 工具列表非空时不注入

- **WHEN** `collectTools()` 返回非空列表
- **THEN** MUST NOT 注入 `_noop` 工具

### Requirement: 结构化输出

当 `Engine.StructuredOutputSchema` 非空时，最后一轮 LLM 调用 MUST 支持结构化输出模式。Engine MUST 在完成所有工具调用后的最后一轮追加结构化输出工具，并将模型的 tool_call 结果作为结构化输出返回。

#### Scenario: 配置了结构化输出

- **WHEN** StructuredOutputSchema 为有效 JSON Schema
- **THEN** 最终响应 MUST 符合该 schema

### Requirement: 重试状态事件

Engine MUST 在每次重试等待前通过 Bus 发布 `session.retry` 事件，payload 包含 session_id、attempt、delay_ms、error 信息。

#### Scenario: 重试时发布事件

- **WHEN** LLM 调用因 RateLimit 进入重试
- **THEN** Bus MUST 收到 `session.retry` 事件
- **AND** payload MUST 包含重试次数和等待时间

## MODIFIED Requirements

### Requirement: 循环终止条件

#### Scenario: context 取消终止

- **WHEN** 循环检测到 `ctx.Err() != nil`
- **THEN** Engine MUST 立即终止循环并持久化已有消息

#### Scenario: 最大轮数优雅终止

- **WHEN** 循环达到 `MaxToolRounds` 次
- **THEN** Engine MUST 在最后一轮注入 MAX_STEPS 警告并禁用工具，让模型生成文本回复
