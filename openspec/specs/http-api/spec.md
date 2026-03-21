# http-api Specification

## Purpose

定义 HTTP API 端点、流式协议、鉴权与错误处理。

## Requirements

### Requirement: 首版提供 HTTP 服务

首版系统 MUST 暴露 HTTP API 服务器，实现远程或编辑器客户端所需的、`agent-runtime`、`llm-and-tools` 与 `persistence` 所要求的路由与行为。服务器 MUST 可通过 `cli-and-config` 规定的 CLI 方式启动。

#### Scenario: 首版提供 HTTP 服务

- **WHEN** 用户使用有效配置启动 HTTP 服务器
- **THEN** 进程 MUST 在配置的地址上打开 TCP 监听，并在优雅关闭前持续处理 HTTP 请求

### Requirement: API 版本前缀

所有 HTTP 路由 MUST 归组在版本前缀下（例如 `/v1`）。破坏兼容的路径或载荷变更 MUST 提升主版本前缀或置于新前缀后；此类变更 MUST 在发行说明中标注 BREAKING。

#### Scenario: 路由位于版本前缀下

- **WHEN** 客户端请求已文档化的 API 端点
- **THEN** 请求路径 MUST 以该发行定义的版本前缀开头

### Requirement: 默认绑定 loopback

默认监听地址 MUST 为 loopback（例如 `127.0.0.1`），并文档化默认端口。绑定所有网卡（`0.0.0.0` 或 `::`）MUST 显式配置，且 MUST 按下文要求启用鉴权。

#### Scenario: 非 loopback 必须鉴权

- **WHEN** 配置的绑定地址不是仅 loopback
- **THEN** 除非已在启动时配置并校验鉴权凭据，否则服务器 MUST NOT 启动

### Requirement: 鉴权基线

系统 MUST 至少支持以下之一：静态 Bearer token 校验，或通过已配置头或查询参数进行的共享密钥校验。鉴权配置 MUST 可通过与 `cli-and-config` 相同的优先级规则加载。

#### Scenario: 需要鉴权时拒绝未认证请求

- **WHEN** 当前绑定模式要求鉴权且请求缺少有效凭据
- **THEN** 服务器 MUST 返回 HTTP 401，且 MUST NOT 执行特权操作

### Requirement: 优雅关闭

服务器 MUST 在 SIGTERM 或平台等效的停止信号下优雅关闭：拒绝新连接，并在可配置的宽限期内完成进行中的请求后退出。

#### Scenario: 关闭时排空

- **WHEN** 收到关闭信号
- **THEN** 监听器 MUST 关闭，进行中的 HTTP handler MUST 在宽限截止时间前得到完成机会

### Requirement: 错误响应

API 错误 MUST 返回 JSON 体，含 `error` 对象（`code` 字符串 + `message` 字符串）。HTTP 状态码 MUST 与语义一致。tool_call 执行失败 MUST NOT 导致 HTTP 错误——失败信息 MUST 作为 tool_result 消息体内的 `isError` 字段返回。

#### Scenario: 工具失败不影响 HTTP 状态码

- **WHEN** ReAct 循环中某工具调用失败
- **THEN** HTTP 响应 MUST 为 200，失败信息 MUST 包含在响应消息列表的 tool_result 消息中

### Requirement: 流式响应选项

若实现通过 HTTP 暴露流式助手输出，MUST 使用 SSE（`text/event-stream`）。SSE 事件从纯 `data: <chunk>` 变更为结构化事件格式 `data: {"type":"text","text":"..."}` / `data: {"type":"tool_call","name":"...","args":{...}}` / `data: {"type":"tool_result","tool_call_id":"...","content":"..."}`。非流式客户端 MUST 仍能通过已文档化的替代方式获得完整响应。

#### Scenario: 结构化 SSE 事件

- **WHEN** 客户端通过 SSE 接收流式补全
- **THEN** 每个 SSE 事件的 `data` MUST 为包含 `type` 字段的 JSON 对象

#### Scenario: 非流式回退

- **WHEN** 客户端不请求流式
- **THEN** 端点 MUST 返回完整 JSON 响应

### Requirement: 会话集合查询

系统 MUST 提供已文档化的 HTTP 端点用于列出会话（至少支持按工作区或项目过滤中的一种）；响应 MUST 为 JSON 且 MUST 包含稳定字段（会话 id、创建时间等）。

#### Scenario: 列表成功

- **WHEN** 客户端请求会话列表且凭据有效
- **THEN** 响应 MUST 为 200 且 MUST 包含会话条目数组或分页包装

### Requirement: 消息分页查询

系统 MUST 提供按会话查询消息历史的端点，且 MUST 支持分页参数（例如 cursor/limit）；消息顺序 MUST 与因果顺序一致。

#### Scenario: 分页参数生效

- **WHEN** 客户端传入分页参数请求消息
- **THEN** 返回集 MUST 受分页约束且 MUST NOT 包含其他会话消息

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

### Requirement: 提供商与模型发现端点

系统 MUST 提供 `GET /v1/providers` 端点返回已注册提供商列表，以及 `GET /v1/providers/{id}/models` 端点返回指定提供商的可用模型列表。

#### Scenario: 列出提供商

- **WHEN** 客户端 GET `/v1/providers`
- **THEN** 响应 MUST 为 JSON 数组，每项包含提供商 `id` 和 `name`

#### Scenario: 列出模型

- **WHEN** 客户端 GET `/v1/providers/openai/models`
- **THEN** 响应 MUST 为 JSON 数组，每项包含模型标识符

### Requirement: 会话 Fork 端点

系统 MUST 提供 `POST /v1/sessions/{id}/fork` 端点，接受 `{message_seq}` 载荷，返回新创建的 fork 会话。

#### Scenario: Fork 成功

- **WHEN** 客户端 POST fork 请求且 message_seq 有效
- **THEN** 响应 MUST 为 201 且包含新会话 ID

### Requirement: 会话 Revert 端点

系统 MUST 提供 `POST /v1/sessions/{id}/revert` 端点，接受 `{message_seq}` 载荷，删除 seq 之后的消息。

#### Scenario: Revert 成功

- **WHEN** 客户端 POST revert 请求且 message_seq 有效
- **THEN** 响应 MUST 为 200

### Requirement: 会话元数据更新端点

系统 MUST 提供 `PATCH /v1/sessions/{id}` 端点，接受 `{title, archived}` 载荷更新会话元数据。

#### Scenario: 更新标题

- **WHEN** 客户端 PATCH `{"title": "新标题"}`
- **THEN** 会话标题 MUST 更新，响应 MUST 为 200

### Requirement: Usage 统计端点

系统 MUST 提供 `GET /v1/sessions/{id}/usage` 端点，返回该会话的 token 使用量聚合。

#### Scenario: 查询 usage

- **WHEN** 客户端 GET usage
- **THEN** 响应 MUST 包含 `prompt_tokens`、`completion_tokens`、`total_tokens` 字段

### Requirement: 配置读取端点

系统 MUST 提供 `GET /v1/config` 端点，返回当前运行时的非敏感配置信息（排除 API key）。

#### Scenario: 读取配置

- **WHEN** 客户端 GET `/v1/config`
- **THEN** 响应 MUST 为 JSON 且 MUST NOT 包含任何 API key 值
