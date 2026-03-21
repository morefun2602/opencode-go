# react-loop Specification

## Purpose

TBD

## Requirements

### Requirement: 消息历史加载

Engine MUST 在每轮对话开始前从持久化层加载当前会话的全部历史消息（含 system、user、assistant、tool 角色），并将其转换为 Provider 可消费的 `[]Message` 格式。

#### Scenario: 历史加载成功

- **WHEN** 用户在已有若干轮对话的会话中发送新消息
- **THEN** Engine MUST 将所有历史消息与新用户消息一并传给 Provider

### Requirement: 工具定义注入

Engine MUST 在调用 Provider 前收集当前可用的全部工具定义（内置 + MCP），并以 `[]ToolDef` 形式传给 Provider。工具收集 MUST 使用 Agent 级别的 ToolFilter 替代纯 Mode Tags 过滤。

#### Scenario: 工具列表包含内置与 MCP 工具

- **WHEN** 会话配置了内置工具与至少一个 MCP 服务端
- **THEN** Provider 收到的 tools 列表 MUST 包含两类工具的定义

#### Scenario: Agent 工具权限过滤

- **WHEN** 当前 Agent 的 ToolPermissions 禁止某工具
- **THEN** 该工具 MUST NOT 出现在 Provider 收到的工具列表中

### Requirement: tool_calls 解析与执行

当 Provider 返回的 `FinishReason` 为 `"tool_calls"` 时，Engine MUST 遍历响应中的每个 tool_call，通过 `ToolRouter` 解析并执行，将结果构建为 `role=tool` 的消息回注到消息列表。

#### Scenario: 模型请求调用已注册工具

- **WHEN** Provider 返回包含一个已知工具名的 tool_call
- **THEN** Engine MUST 执行该工具并将结果消息追加到消息历史

#### Scenario: 模型请求调用未知工具

- **WHEN** Provider 返回包含未注册工具名的 tool_call
- **THEN** Engine MUST 将包含错误信息的 tool_result 消息追加到消息历史，且 MUST NOT 中断循环

### Requirement: 循环终止条件

Engine MUST 在以下任一条件满足时终止 ReAct 循环：（1）Provider 返回 `FinishReason != "tool_calls"`；（2）循环轮数达到 `MaxToolRounds` 上限；（3）doom loop 检测触发且用户选择终止；（4）ContextOverflow 压缩后仍失败；（5）context 被取消。

#### Scenario: 最大轮数保护

- **WHEN** 循环达到 `MaxToolRounds`（默认 25）次且 Provider 仍返回 tool_calls
- **THEN** Engine MUST 终止循环并将当前助手消息作为最终响应

#### Scenario: 最大轮数优雅终止

- **WHEN** 循环达到 `MaxToolRounds` 次
- **THEN** Engine MUST 在最后一轮注入 MAX_STEPS 警告并禁用工具，让模型生成文本回复

#### Scenario: Doom loop 用户终止

- **WHEN** doom loop 检测触发且用户选择终止
- **THEN** Engine MUST 立即终止循环

#### Scenario: 压缩失败终止

- **WHEN** ContextOverflow 触发压缩后重试仍失败
- **THEN** Engine MUST 终止循环并返回上下文溢出错误

#### Scenario: context 取消终止

- **WHEN** 循环检测到 `ctx.Err() != nil`
- **THEN** Engine MUST 立即终止循环并持久化已有消息

### Requirement: 消息持久化

Engine MUST 在一轮对话（含所有中间 tool_call/result）结束后将全部新增消息原子写入持久化层。

#### Scenario: 含工具调用的完整轮次

- **WHEN** 一轮对话产生用户消息、助手消息（含 tool_calls）、tool_result 消息、最终助手消息共 N 条
- **THEN** 持久化层 MUST 包含全部 N 条新消息且顺序 MUST 与因果顺序一致

### Requirement: Doom loop 检测

Engine MUST 在 ReAct 循环中维护最近 N 次（默认 3）tool_call 的签名（工具名 + 参数内容 hash）。当连续 N 次签名完全相同时，Engine MUST 通过 Permission.Ask 通知用户并询问是否继续，而非直接终止。

#### Scenario: 检测到 doom loop

- **WHEN** 连续 3 次 tool_call 的工具名和参数 hash 完全相同
- **THEN** Engine MUST 暂停循环并通过 Permission.Ask 向用户展示重复信息，等待用户决定继续或终止

#### Scenario: 相同工具不同参数不触发

- **WHEN** 连续 3 次调用同一工具但参数不同
- **THEN** Engine MUST NOT 视为 doom loop

### Requirement: ContextOverflow 错误恢复

当 LLM Provider 返回 ContextOverflow 类型错误时，Engine MUST 触发上下文压缩流程（Compaction），而非终止循环。压缩完成后 MUST 自动重试当前轮次。

#### Scenario: 上下文溢出触发压缩

- **WHEN** Provider 返回 ContextOverflow 错误
- **THEN** Engine MUST 调用 Compaction.Process() 压缩历史，然后重试 LLM 调用

#### Scenario: 压缩后仍然溢出

- **WHEN** 压缩后重试仍返回 ContextOverflow
- **THEN** Engine MUST 终止循环并返回错误

### Requirement: Permission/Question Rejected 处理

当工具执行过程中 Permission 或 Question 被用户拒绝时，Engine MUST 将对应 tool_result 标记为 blocked 状态，并将拒绝信息作为工具结果回注，而非终止整个循环。

#### Scenario: Permission 被拒绝

- **WHEN** 用户拒绝某工具的权限请求
- **THEN** Engine MUST 将 "permission denied" 作为该工具的结果回注消息历史，循环继续

### Requirement: Compaction 会话集成

Engine MUST 在 ReAct 循环中集成完整的 Compaction 流程：每轮结束后检查 isOverflow，溢出时依次执行 prune 和 process。

#### Scenario: 每轮结束后检查溢出

- **WHEN** 一轮 LLM 调用完成并返回 token 使用量
- **THEN** Engine MUST 调用 IsOverflow() 检查是否需要压缩

#### Scenario: 溢出触发 prune + process

- **WHEN** IsOverflow() 返回 true 且 config.compaction.auto 为 true
- **THEN** Engine MUST 先调用 Prune() 裁剪旧 tool 输出，再调用 Process() 使用 compaction agent 生成摘要

#### Scenario: compaction 配置为关闭

- **WHEN** config.compaction.auto 为 false
- **THEN** Engine MUST NOT 自动触发 compaction

### Requirement: IsOverflow 基于 Token

Engine MUST 实现基于 token 使用量的溢出检测：当 token 总量 ≥ 模型上下文限制减去保留量（config.compaction.reserved，默认 20000）时判定为溢出。

#### Scenario: token 未达限制

- **WHEN** token 使用量为 50000 且模型限制为 128000
- **THEN** IsOverflow MUST 返回 false

#### Scenario: token 超过限制

- **WHEN** token 使用量为 110000 且模型限制为 128000 且 reserved 为 20000
- **THEN** IsOverflow MUST 返回 true

### Requirement: Prune 裁剪旧 Tool 输出

Engine MUST 实现 Prune 逻辑：从最旧的消息开始，将 tool_result 的 content 替换为 `[pruned]` 标记，保留最近约 40000 tokens 的 tool 结果不被裁剪。skill 工具的结果 MUST 不被裁剪。

#### Scenario: 裁剪旧 tool 输出

- **WHEN** 消息列表包含 20 个 tool_result 且总 token 超限
- **THEN** 旧的 tool_result content MUST 被替换为 `[pruned]`，仅保留最近的结果

#### Scenario: skill 工具结果受保护

- **WHEN** Prune 处理到 skill 工具的 tool_result
- **THEN** 该结果 MUST NOT 被裁剪

### Requirement: 系统提示集成

Engine MUST 在构建消息列表时，通过 `prompt.Build()` 构建系统提示（而非直接使用 Engine.SystemPrompt 字符串）。系统提示 MUST 包含模型/Agent base prompt、环境信息、InstructionPrompt 和技能列表。

#### Scenario: 使用 prompt 模块构建系统提示

- **WHEN** Engine 开始新一轮 CompleteTurn
- **THEN** 系统消息 MUST 由 prompt.Build() 生成，包含环境信息和技能列表

### Requirement: Provider Router 集成

Engine MUST 通过 `llm.Router` 获取 Provider 和模型 ID，而非直接使用 `e.LLM` 字段。不同类型的任务（普通会话、compaction、title、summary）MUST 使用不同的模型。

#### Scenario: 普通会话使用默认模型

- **WHEN** Engine 执行普通用户会话
- **THEN** MUST 通过 Router.DefaultModel() 获取 Provider

#### Scenario: 内部任务使用小模型

- **WHEN** Engine 执行 compaction 任务
- **THEN** MUST 通过 Router.SmallModel() 获取 Provider

### Requirement: Snapshot 集成点

Engine MUST 在每个 ReAct step 开始前调用 Snapshot.Track()，在 step 完成后调用 Snapshot.Patch()，记录该步骤的文件变更。当 Snapshot 模块不可用时 MUST 静默跳过。

#### Scenario: Step 级别快照

- **WHEN** 一个包含文件写操作的 tool_call 执行完成
- **THEN** Engine MUST 记录该 step 的文件变更 patch

#### Scenario: Snapshot 不可用时跳过

- **WHEN** Snapshot 模块因非 git 目录而不可用
- **THEN** Engine MUST 跳过快照操作且 MUST NOT 报错

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
