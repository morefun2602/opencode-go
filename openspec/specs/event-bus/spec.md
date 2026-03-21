# event-bus Specification

## Purpose

定义进程内类型安全的事件总线与 SSE 事件推送端点。

## Requirements

### Requirement: 进程内 Pub/Sub

系统 MUST 提供类型安全的进程内事件总线（`internal/bus`），支持发布和订阅事件。事件类型 MUST 包括：`session.created`、`session.updated`、`message.created`、`tool.start`、`tool.end`、`permission.ask`、`question.ask`。

#### Scenario: 发布与订阅

- **WHEN** 代码发布 `session.created` 事件
- **THEN** 所有已注册的该事件订阅者 MUST 收到通知

#### Scenario: 无订阅者时不阻塞

- **WHEN** 发布事件但无订阅者
- **THEN** 发布操作 MUST 立即返回，不阻塞

### Requirement: SSE 事件端点

系统 MUST 提供 `GET /v1/events` SSE 端点，将事件总线中的事件序列化为 SSE 格式推送给客户端。每个 SSE 事件 MUST 包含 `type` 和 `data`（JSON）字段。

#### Scenario: 客户端接收事件

- **WHEN** 客户端建立 SSE 连接后，有新会话创建
- **THEN** 客户端 MUST 收到 `event: session.created` 的 SSE 事件

#### Scenario: 连接断开清理

- **WHEN** 客户端断开 SSE 连接
- **THEN** 系统 MUST 取消该连接的订阅，不泄漏 goroutine
