## ADDED Requirements

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

### Requirement: Snapshot 集成点

Engine MUST 在每个 ReAct step 开始前调用 Snapshot.Track()，在 step 完成后调用 Snapshot.Patch()，记录该步骤的文件变更。当 Snapshot 模块不可用时 MUST 静默跳过。

#### Scenario: Step 级别快照

- **WHEN** 一个包含文件写操作的 tool_call 执行完成
- **THEN** Engine MUST 记录该 step 的文件变更 patch

#### Scenario: Snapshot 不可用时跳过

- **WHEN** Snapshot 模块因非 git 目录而不可用
- **THEN** Engine MUST 跳过快照操作且 MUST NOT 报错

## MODIFIED Requirements

### Requirement: 循环终止条件

Engine MUST 在以下任一条件满足时终止 ReAct 循环：（1）Provider 返回 `FinishReason != "tool_calls"`；（2）循环轮数达到 `MaxToolRounds` 上限；（3）doom loop 检测触发且用户选择终止；（4）ContextOverflow 压缩后仍失败。

#### Scenario: 最大轮数保护

- **WHEN** 循环达到 `MaxToolRounds`（默认 25）次且 Provider 仍返回 tool_calls
- **THEN** Engine MUST 终止循环并将当前助手消息作为最终响应

#### Scenario: Doom loop 用户终止

- **WHEN** doom loop 检测触发且用户选择终止
- **THEN** Engine MUST 立即终止循环

#### Scenario: 压缩失败终止

- **WHEN** ContextOverflow 触发压缩后重试仍失败
- **THEN** Engine MUST 终止循环并返回上下文溢出错误
