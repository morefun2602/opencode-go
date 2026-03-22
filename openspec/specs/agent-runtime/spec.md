# agent-runtime Specification

## Purpose

定义智能体运行时的会话管理、消息编排与 ReAct 循环行为。

## Requirements

### Requirement: 会话生命周期

系统 MUST 支持创建、选择与关闭会话。每个会话在进程存活期内或显式关闭前具有稳定标识符，同进程内活跃会话的标识符 MUST 唯一。Engine.CompleteTurn 从「单次 LLM 调用」变更为「ReAct 循环」：每次调用 MUST 维护消息历史（从 Store 加载），构建 system prompt（含技能注入），收集工具定义，进入循环直到模型不再请求工具或达到上限。

#### Scenario: 新会话获得标识符

- **WHEN** 客户端通过支持的接口（CLI 或 HTTP API）请求新会话
- **THEN** 系统 MUST 返回非空会话标识符，并在关闭前将该会话与后续操作关联

#### Scenario: CompleteTurn 执行 ReAct 循环

- **WHEN** 调用 CompleteTurn 时会话已有历史消息且工具可用
- **THEN** Engine MUST 加载历史、注入 system prompt 与工具列表、调用 Provider、解析 tool_calls、执行工具、回注结果，循环直至终止条件满足

#### Scenario: CompleteTurn 使用活跃 Agent

- **WHEN** 调用 CompleteTurn 时 session 有 Agent 覆盖
- **THEN** Engine MUST 使用该 Agent 构建系统提示、收集工具、选择模型
- **AND** MUST NOT 使用固定的 `e.Agent`

### Requirement: 上下文取消

包含模型轮次与工具调用的长时操作 MUST 遵守 `context.Context` 取消：当上下文被取消时，系统 MUST 停止为该会话当前轮次调度新工作，并在文档化的关闭时限内向调用方返回取消错误。

#### Scenario: 取消后停止工作

- **WHEN** 调用方取消进行中的智能体操作所关联的上下文
- **THEN** 系统在观测到取消后 MUST NOT 再为该操作输出助手内容，并将取消传递给调用方

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

### Requirement: 压缩与摘要（Compaction）

系统 MUST 支持可配置的对话压缩或摘要策略，使长会话在超过阈值时 MUST 能生成摘要或裁剪上下文且 MUST 保持用户意图可追溯。

#### Scenario: 超长会话触发策略

- **WHEN** 会话 token 或消息条数超过配置阈值
- **THEN** 系统 MUST 应用压缩或摘要且 MUST 继续允许新 turn

### Requirement: Step Summary 生命周期

系统 MUST 在 ReAct step 生命周期接入增量摘要能力：每轮结束后更新当前 session 的 step summary，并向可观测层发布摘要事件。

#### Scenario: step 完成后写入摘要

- **WHEN** 某轮 step 完成（无论有无 tool_calls）
- **THEN** 系统 MUST 为该 step 生成/更新摘要条目
- **AND** Bus MUST 发布 `session.summary` 事件

### Requirement: 重试与回退（Retry / Revert）

系统 MUST 为模型或工具失败提供可配置的重试策略；在支持的场景下 MUST 提供回退到先前消息状态的能力（与 `persistence` 协同），且 MUST NOT 在无确认时丢弃用户数据。

#### Scenario: 重试次数上限

- **WHEN** 连续失败达到配置上限
- **THEN** 系统 MUST 停止重试并 MUST 向用户或客户端报告

### Requirement: 结构化输出模式

当配置或模型支持结构化输出时，系统 MUST 校验输出是否符合声明的 schema；校验失败 MUST 反馈给编排层且 MUST NOT 当作成功完成。

#### Scenario: schema 校验失败

- **WHEN** 模型返回不符合 schema 的 JSON
- **THEN** 系统 MUST 返回校验错误且 MUST NOT 持久化为成功助手消息

### Requirement: 消息模型版本

系统 MUST 为消息存储与 API 暴露版本或 `schema` 字段，以支持与上游 message v2 等模型的分阶段对齐；旧客户端 MUST 在未升级时仍能获得向后兼容视图或明确 BREAKING 说明。

#### Scenario: 版本字段存在

- **WHEN** 客户端读取消息资源
- **THEN** 响应 MUST 包含版本或模型标识字段或文档化等价物

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

### Requirement: Engine Agent 感知 — collectTools

Engine 的 `collectTools()` MUST 在每轮循环开始时查询当前 session 的活跃 Agent（通过 `ModeSwitch` 或 session-level override），而非始终使用固定的 `e.Agent`。活跃 Agent 变化时，工具列表 MUST 立即反映。

#### Scenario: session 级 Agent 覆盖

- **WHEN** session 通过 `plan_enter` 切换到 plan Agent
- **THEN** `collectTools()` 在该 session 的后续轮次 MUST 使用 plan Agent 的 Permission 过滤工具

#### Scenario: 无覆盖时使用默认

- **WHEN** session 无 Agent 覆盖
- **THEN** `collectTools()` MUST 使用 `e.Agent`（通常为 AgentBuild）

### Requirement: Engine Agent 感知 — 模型选择

Engine MUST 在执行 LLM 调用前检查当前活跃 Agent 的 `Model` 字段。若非空，MUST 通过 `Router` 获取对应 Provider 和模型 ID 替代默认 LLM。

#### Scenario: Agent 指定模型

- **WHEN** 活跃 Agent 的 Model 为 "openai/gpt-4o"
- **THEN** Engine MUST 使用 OpenAI Provider 的 gpt-4o 模型执行 LLM 调用

#### Scenario: Agent 未指定模型

- **WHEN** 活跃 Agent 的 Model 为空
- **THEN** Engine MUST 使用 `e.LLM` 或 `Router.DefaultModel()` 执行 LLM 调用

### Requirement: Engine Agent 感知 — 轮数限制

Engine MUST 在确定 `maxRounds` 时优先使用活跃 Agent 的 `Steps` 字段（若 > 0），否则使用 `e.MaxToolRounds`。

#### Scenario: Agent Steps 覆盖

- **WHEN** 活跃 Agent 的 Steps 为 10 且 Engine MaxToolRounds 为 25
- **THEN** 循环最多执行 10 轮

### Requirement: Engine Agent 感知 — buildSystemPrompt

`buildSystemPrompt()` MUST 使用活跃 Agent（而非固定 `e.Agent`）的 Prompt 字段。当 session 切换到不同 Agent 时，系统提示 MUST 反映新 Agent 的 Prompt。

#### Scenario: plan Agent 有 Prompt

- **WHEN** 活跃 Agent 为 plan 且 plan Agent 定义了自定义 Prompt
- **THEN** buildSystemPrompt MUST 使用 plan Agent 的 Prompt 替代 provider base prompt

### Requirement: session 级 Agent 管理

Engine MUST 维护 session → Agent 的映射（与现有 `ModeSwitch` 合并或替代）。提供 `GetSessionAgent(sessionID) Agent` 和 `SetSessionAgent(sessionID, Agent)` 方法。session 结束时 MUST 清理映射。

#### Scenario: session 结束清理

- **WHEN** CompleteTurn 执行完毕
- **THEN** session Agent 映射 MUST NOT 泄漏（defer 清理）

### Requirement: Confirm 函数注入

Engine 的 `Confirm` 字段 MUST 在 `wireEngine` 中被正确设置。当 Policy 要求 `ask` 且 `Confirm` 为 nil 时，MUST 默认拒绝。

#### Scenario: Confirm 在 REPL 中

- **WHEN** REPL 模式运行且工具权限为 ask
- **THEN** Engine.Confirm MUST 调用终端交互等待用户输入 y/n

#### Scenario: Confirm 在 HTTP 中

- **WHEN** HTTP API 模式运行
- **THEN** Engine.Confirm MUST 通过 Permission/Question 机制异步等待客户端响应
