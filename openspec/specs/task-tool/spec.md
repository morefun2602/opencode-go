# task-tool Specification

## Purpose

TBD

## Requirements

### Requirement: 子 agent 执行

task 工具 MUST 接受任务描述文本，创建独立子会话，在子会话中运行完整的 ReAct 循环（复用 Engine），并返回子 agent 的最终文本输出。

#### Scenario: 子任务成功

- **WHEN** 调用 task 工具并传入合法任务描述
- **THEN** 系统 MUST 创建子会话、运行至模型完成、返回助手最终文本

### Requirement: 嵌套深度限制

task 工具 MUST 支持可配置的最大嵌套深度（默认 2），超过深度时 MUST 拒绝创建子 agent 并返回错误。

#### Scenario: 深度超限

- **WHEN** 子 agent 内部再次调用 task 工具且已达最大深度
- **THEN** 工具 MUST 返回错误提示"超过最大嵌套深度"且 MUST NOT 创建新子会话

### Requirement: 子会话隔离

子 agent 的消息历史 MUST 存储在独立的子会话中（不同 sessionID），且 MUST NOT 污染父会话的消息历史。

#### Scenario: 父子会话独立

- **WHEN** 子 agent 执行完成
- **THEN** 父会话的消息历史 MUST 仅包含 task tool_call 与 tool_result，不含子会话的中间消息
