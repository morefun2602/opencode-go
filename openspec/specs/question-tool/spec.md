# question-tool Specification

## Purpose

定义 question 工具的提问交互机制，支持 CLI 与 HTTP 两种模式下的用户问答流程。

## Requirements

### Requirement: 向用户提问

question 工具 MUST 接受 `question` 和可选的 `options` 参数，暂停 ReAct 循环等待用户回复，并将回复作为工具结果返回。

#### Scenario: CLI 模式提问

- **WHEN** 在 REPL/TUI 模式下模型调用 question
- **THEN** 系统 MUST 在终端显示问题文本与选项，等待用户输入，将输入作为 tool_result 返回

#### Scenario: HTTP 模式异步提问

- **WHEN** 在 HTTP 模式下模型调用 question
- **THEN** 系统 MUST 通过事件总线发射 question 事件，阻塞等待 `POST /v1/question/reply` 回复，超时后返回超时错误

### Requirement: 回复端点

系统 MUST 提供 `POST /v1/question/reply` 端点，接受 `{question_id, answer}` 载荷，将回复传递给等待中的 question 工具。

#### Scenario: 回复匹配

- **WHEN** 客户端 POST 回复且 question_id 匹配当前等待的提问
- **THEN** 系统 MUST 将 answer 传递给阻塞中的工具调用并恢复 ReAct 循环

#### Scenario: 未知 question_id

- **WHEN** 客户端 POST 的 question_id 不匹配任何等待中的提问
- **THEN** 系统 MUST 返回 404
