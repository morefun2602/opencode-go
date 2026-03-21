## MODIFIED Requirements

### Requirement: subagent_type Agent 选择

task 工具 MUST 根据 `subagent_type` 参数选择对应的 Agent。`subagent_type` 为有效 Agent 名称时，子 Engine MUST 使用该 Agent 的 Mode、Permission、Prompt、Model 和 Steps。

#### Scenario: subagent_type 为 explore

- **WHEN** task 工具被调用且 subagent_type 为 "explore"
- **THEN** 子 Engine MUST 使用 AgentExplore
- **AND** 子 Engine 的工具列表 MUST 仅包含 read 标签和 bash 工具
- **AND** 子 Engine 的系统提示 MUST 使用 explore Agent 的 Prompt

#### Scenario: subagent_type 为 general

- **WHEN** task 工具被调用且 subagent_type 为 "general"
- **THEN** 子 Engine MUST 使用 AgentGeneral
- **AND** 子 Engine 的工具列表 MUST 排除 todowrite

#### Scenario: subagent_type 为空或未知

- **WHEN** task 工具被调用且 subagent_type 为空
- **THEN** 子 Engine MUST 使用 AgentGeneral（默认子 Agent）

- **WHEN** subagent_type 为未知名称（如 "nonexistent"）
- **THEN** task 工具 MUST 返回错误列出可用 Agent

### Requirement: 子 Engine Agent 传递

task 工具创建子 Engine 时，MUST 将选定的 Agent 传递给子 Engine。子 Engine 的 `Agent` 字段 MUST 为选定 Agent，而非主 Engine 的 Agent。

#### Scenario: 子 Engine 独立 Agent

- **WHEN** 主 Engine 使用 AgentBuild 且 task 指定 subagent_type 为 "explore"
- **THEN** 子 Engine.Agent MUST 为 AgentExplore
- **AND** 子 Engine 的 collectTools MUST 使用 AgentExplore 的过滤规则

### Requirement: 可调用 Agent 过滤

task 工具 MUST 过滤可调用的 Agent 列表：Hidden Agent MUST NOT 可被 task 调用。Primary Agent（build、plan）MUST NOT 作为子 Agent 使用，仅 subagent 模式的 Agent（general、explore、自定义非 primary）可用。

#### Scenario: 不可调用 hidden Agent

- **WHEN** task 工具的 subagent_type 为 "compaction"
- **THEN** task 工具 MUST 返回错误（compaction 为 hidden Agent）

#### Scenario: 不可调用 primary Agent

- **WHEN** task 工具的 subagent_type 为 "build"
- **THEN** task 工具 MUST 返回错误（build 为 primary Agent）

### Requirement: TaskRunner 接口增强

`TaskRunner` 接口 MUST 支持传递 Agent 信息给子 Engine。新增 `CompleteTurnWithAgent(ctx, workspaceID, sessionID, userText string, agent Agent) (string, error)` 方法，或通过 context 传递 Agent。

#### Scenario: 子 Engine 使用指定 Agent

- **WHEN** TaskRunner.CompleteTurnWithAgent 被调用且 agent 为 AgentExplore
- **THEN** 子 Engine 的 ReAct 循环 MUST 使用 AgentExplore 的所有属性
