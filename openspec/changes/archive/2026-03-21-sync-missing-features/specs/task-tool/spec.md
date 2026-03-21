## ADDED Requirements

### Requirement: task_id 恢复机制

task 工具 MUST 支持可选的 `task_id` 参数。当提供 task_id 时，系统 MUST 查找已有的子会话并在该会话中继续对话，而非创建新会话。找不到对应会话时 MUST 返回错误。

#### Scenario: 恢复已有子会话

- **WHEN** 调用 task 工具并传入有效的 task_id
- **THEN** 系统 MUST 在对应的已有子会话中追加新消息并执行 ReAct 循环

#### Scenario: task_id 无效

- **WHEN** 调用 task 工具并传入不存在的 task_id
- **THEN** 系统 MUST 返回错误提示 task_id 不存在

#### Scenario: 新任务返回 task_id

- **WHEN** 调用 task 工具未传入 task_id（创建新任务）
- **THEN** 工具返回结果 MUST 包含新创建的 task_id 供后续恢复使用

### Requirement: subagent_type 参数

task 工具 MUST 支持可选的 `subagent_type` 参数，用于指定子 agent 使用的模式类型（如 build、plan、explore 或自定义）。未指定时 MUST 使用默认模式。

#### Scenario: 指定子 agent 类型

- **WHEN** 调用 task 工具并指定 subagent_type 为 "plan"
- **THEN** 子 agent MUST 以 plan 模式运行（仅包含 plan 模式的工具）

#### Scenario: 无效 subagent_type

- **WHEN** 指定了不存在的 subagent_type
- **THEN** 系统 MUST 返回错误列出可用类型

### Requirement: description 参数

task 工具 MUST 支持可选的 `description` 参数，提供子任务的简要描述。该描述 MUST 存储在子会话元数据中用于追踪。

#### Scenario: 描述存储

- **WHEN** 调用 task 工具并传入 description
- **THEN** 子会话的元数据 MUST 包含该描述

## MODIFIED Requirements

### Requirement: 子 agent 执行

task 工具 MUST 接受任务 prompt 文本，根据是否提供 task_id 决定创建新子会话或恢复已有子会话，在子会话中运行完整的 ReAct 循环（复用 Engine），并返回子 agent 的最终文本输出和 task_id。

#### Scenario: 子任务成功

- **WHEN** 调用 task 工具并传入合法任务描述
- **THEN** 系统 MUST 创建子会话、运行至模型完成、返回助手最终文本和 task_id

#### Scenario: 恢复子任务成功

- **WHEN** 调用 task 工具并传入有效 task_id 和新 prompt
- **THEN** 系统 MUST 在已有子会话中追加消息、运行循环、返回最终文本
