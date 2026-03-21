# Capability: agent-runtime (delta)

## MODIFIED Requirements

### Requirement: 会话生命周期

系统 MUST 支持创建、选择与关闭会话。每个会话在进程存活期内或显式关闭前具有稳定标识符，同进程内活跃会话的标识符 MUST 唯一。Engine.CompleteTurn 从「单次 LLM 调用」变更为「ReAct 循环」：每次调用 MUST 维护消息历史（从 Store 加载），构建 system prompt（含技能注入），收集工具定义，进入循环直到模型不再请求工具或达到上限。

#### Scenario: 新会话获得标识符

- **WHEN** 客户端通过支持的接口（CLI 或 HTTP API）请求新会话
- **THEN** 系统 MUST 返回非空会话标识符，并在关闭前将该会话与后续操作关联

#### Scenario: CompleteTurn 执行 ReAct 循环

- **WHEN** 调用 CompleteTurn 时会话已有历史消息且工具可用
- **THEN** Engine MUST 加载历史、注入 system prompt 与工具列表、调用 Provider、解析 tool_calls、执行工具、回注结果，循环直至终止条件满足

### Requirement: 消息顺序

在同一会话内，同一对话线程上的用户、助手、工具消息在启用持久化时 MUST 按因果顺序写入：对某一 turn，助手内容不得记录在其所回复的用户消息之前；工具结果消息 MUST 排在触发它的助手 tool_call 消息之后。

#### Scenario: 含工具调用的有序持久化

- **WHEN** 同一次 turn 内产生用户消息→助手消息（含 tool_calls）→tool_result→最终助手消息
- **THEN** 持久化 MUST 保持 user → assistant(tool_calls) → tool → assistant(final) 的因果顺序

### Requirement: 可观测进度

系统 MUST 对主要生命周期事件输出结构化日志信号：会话开始、轮次开始、轮次结束、工具调用开始、工具调用结束、致命错误。ReAct 循环的每轮迭代（loop round）MUST 记录当前轮数。事件键名 MUST 在同一小版本系列内保持稳定。

#### Scenario: 记录 ReAct 循环轮数

- **WHEN** ReAct 循环进入第 N 轮
- **THEN** 日志记录 MUST 包含轮数序号与会话标识字段

## ADDED Requirements

### Requirement: system prompt 技能注入

Engine MUST 在构建 system prompt 时调用技能加载模块，将已发现的技能内容拼入 system prompt。

#### Scenario: 技能目录存在

- **WHEN** 配置了 `skills_dir` 且目录中包含 `.md` 文件
- **THEN** system prompt MUST 包含这些文件的内容

### Requirement: 压缩检查

Engine MUST 在每轮结束后检查消息历史长度是否超过 `CompactionTurns` 阈值，超过时 MUST 记录压缩日志（初期可仅记录日志，不实际压缩）。

#### Scenario: 超过压缩阈值

- **WHEN** 会话消息数超过 `CompactionTurns`
- **THEN** 系统 MUST 输出至少一条包含 "compaction" 关键字的日志
