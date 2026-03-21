## ADDED Requirements

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

## MODIFIED Requirements

### Requirement: 工具定义注入

Engine MUST 在调用 Provider 前收集当前可用的全部工具定义（内置 + MCP），并以 `[]ToolDef` 形式传给 Provider。工具收集 MUST 使用 Agent 级别的 ToolFilter 替代纯 Mode Tags 过滤。

#### Scenario: 工具列表包含内置与 MCP 工具

- **WHEN** 会话配置了内置工具与至少一个 MCP 服务端
- **THEN** Provider 收到的 tools 列表 MUST 包含两类工具的定义

#### Scenario: Agent 工具权限过滤

- **WHEN** 当前 Agent 的 ToolPermissions 禁止某工具
- **THEN** 该工具 MUST NOT 出现在 Provider 收到的工具列表中
