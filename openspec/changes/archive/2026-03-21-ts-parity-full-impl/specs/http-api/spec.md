# Capability: http-api (delta)

## MODIFIED Requirements

### Requirement: 流式响应选项

若实现通过 HTTP 暴露流式助手输出，MUST 使用 SSE（`text/event-stream`）。SSE 事件从纯 `data: <chunk>` 变更为结构化事件格式 `data: {"type":"text","text":"..."}` / `data: {"type":"tool_call","name":"...","args":{...}}` / `data: {"type":"tool_result","tool_call_id":"...","content":"..."}`。非流式客户端 MUST 仍能通过已文档化的替代方式获得完整响应。**BREAKING**

#### Scenario: 结构化 SSE 事件

- **WHEN** 客户端通过 SSE 接收流式补全
- **THEN** 每个 SSE 事件的 `data` MUST 为包含 `type` 字段的 JSON 对象

#### Scenario: 非流式回退

- **WHEN** 客户端不请求流式
- **THEN** 端点 MUST 返回完整 JSON 响应

### Requirement: 错误响应

API 错误 MUST 返回 JSON 体，含 `error` 对象（`code` 字符串 + `message` 字符串）。HTTP 状态码 MUST 与语义一致。tool_call 执行失败 MUST NOT 导致 HTTP 错误——失败信息 MUST 作为 tool_result 消息体内的 `isError` 字段返回。

#### Scenario: 工具失败不影响 HTTP 状态码

- **WHEN** ReAct 循环中某工具调用失败
- **THEN** HTTP 响应 MUST 为 200，失败信息 MUST 包含在响应消息列表的 tool_result 消息中

## ADDED Requirements

### Requirement: complete 响应格式变更

`/v1/sessions/{id}/complete` 非流式响应 MUST 从 `{"reply":"..."}` 变更为 `{"messages":[...]}` 格式，返回本轮新增的所有消息（含中间 tool_call/result）。**BREAKING**

#### Scenario: 响应包含全部新增消息

- **WHEN** 一轮对话产生 4 条新消息（user + assistant + tool + assistant）
- **THEN** 非流式响应体 MUST 包含 `messages` 数组且长度为 4

### Requirement: OpenAPI 端点

系统 MUST 提供 `GET /v1/openapi.json` 端点返回 OpenAPI 3 规范文档。

#### Scenario: 获取 OpenAPI

- **WHEN** 客户端 GET `/v1/openapi.json`
- **THEN** 响应 MUST 为 HTTP 200 且 Content-Type 为 `application/json`

### Requirement: ACP 会话事件端点

系统 MUST 提供 `POST /v1/acp/session/event` 端点，接收 ACP 会话事件并桥接至 Store 层。

#### Scenario: ACP 事件接收

- **WHEN** 客户端 POST 合法 ACP 会话事件到该端点
- **THEN** 系统 MUST 返回 HTTP 200 并将事件处理结果写入日志
