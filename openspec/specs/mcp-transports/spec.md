# mcp-transports Specification

## Purpose

TBD

## Requirements

### Requirement: stdio Transport

系统 MUST 提供 MCP stdio 传输实现，通过 `exec.Command` 启动子进程并在 stdin/stdout 上运行 JSON-RPC 2.0 通信。

#### Scenario: 子进程启动与工具发现

- **WHEN** 配置中存在 `transport: "stdio"` 的 MCP 服务端条目
- **THEN** 系统 MUST 启动指定命令为子进程，建立 JSON-RPC 通信，并成功拉取工具列表

#### Scenario: 子进程生命周期管理

- **WHEN** `Client.Close` 被调用或 context 取消
- **THEN** 系统 MUST 终止子进程（SIGTERM + 等待超时后 SIGKILL）且 MUST NOT 泄漏进程

### Requirement: SSE Transport

系统 MUST 提供 MCP SSE 传输实现（旧版远程协议），通过 HTTP GET 建立 SSE 长连接接收服务端事件，通过 HTTP POST 发送 JSON-RPC 请求。

#### Scenario: 连接与 endpoint 事件

- **WHEN** 客户端 GET 服务端 SSE 端点
- **THEN** 系统 MUST 等待 `endpoint` 事件以获取 POST 目标 URL，超时未收到 MUST 返回错误

#### Scenario: 断线重连

- **WHEN** SSE 连接意外断开
- **THEN** 系统 MUST 自动重连（可配置最大重试次数），超过上限 MUST 返回错误

### Requirement: Streamable HTTP Transport

系统 MUST 提供 MCP Streamable HTTP 传输实现（2025-03 规范），通过 POST 到单一端点发送 JSON-RPC 请求，支持服务端以 `application/json` 或 `text/event-stream` 格式返回响应。

#### Scenario: JSON 响应

- **WHEN** 服务端对 POST 请求返回 `application/json`
- **THEN** 系统 MUST 解析该 JSON 作为 JSON-RPC 响应

#### Scenario: SSE 流式响应

- **WHEN** 服务端对 POST 请求返回 `text/event-stream`
- **THEN** 系统 MUST 从 SSE 流中读取 JSON-RPC 响应事件并按 `id` 关联

#### Scenario: 有状态会话

- **WHEN** 服务端返回 `Mcp-Session-Id` header
- **THEN** 后续请求 MUST 携带该 header 以维持会话状态

### Requirement: 传输自动推断

当配置未显式指定 `transport` 字段时，系统 MUST 按以下规则推断传输类型：有 `command` 字段时使用 stdio；有 `url` 字段时使用 streamable_http。

#### Scenario: 自动推断 stdio

- **WHEN** 配置条目有 `command` 无 `transport`
- **THEN** 系统 MUST 使用 stdio 传输

#### Scenario: 自动推断 streamable_http

- **WHEN** 配置条目有 `url` 无 `transport`
- **THEN** 系统 MUST 使用 streamable_http 传输（优先新协议）

### Requirement: 统一 Transport 接口

三种传输 MUST 实现同一 `Transport` 接口，`Client` 层 MUST NOT 包含传输类型特定的逻辑。

#### Scenario: Client 透明切换

- **WHEN** 将同一 MCP 服务端的 `transport` 从 `stdio` 改为 `streamable_http`
- **THEN** Client 的 `Connect` / `CallTool` / `Close` 行为 MUST 保持一致（仅底层通信方式不同）
