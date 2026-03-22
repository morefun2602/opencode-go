## Why

上一轮变更（`opencode-ts-parity-gap`）为 Go 版本建立了骨架包与接口定义，但 **核心运行循环仍然不可用**：`Engine.CompleteTurn` 直接把用户文本转发给 `LLM.Complete`，没有消息历史、没有工具列表注入、没有 tool_calls 解析 / 执行 / 回注循环。同时，LLM 层仅有 stub（echo），MCP 仅有 `NullTransport`，技能未注入，消息模型极简——这意味着 Go 版本在行为面上与上游 TypeScript `packages/opencode` 之间存在 **不可跳过的实现鸿沟**，而非仅仅缺少骨架。

本变更的目标是 **将骨架填充为可运行的、端到端可验证的实现**，使 Go 版本能真正连接模型、调用工具、管理消息历史，并在 CLI/HTTP 上暴露与上游语义等价的行为。

## What Changes

- **实现 ReAct 工具调用循环**：Engine 需维护消息历史（含 system prompt、tool results），解析模型返回的 tool_calls，通过 ToolRouter 执行，回注结果，循环直到模型不再请求工具或达到上限。
- **接入真实 LLM 提供商**：至少实现 OpenAI 与 Anthropic（均通过其 HTTP API），支持流式与非流式，支持 tool_use / function_calling 协议。
- **实现三种 MCP Transport**：stdio（本地子进程）、SSE（旧版远程协议）、Streamable HTTP（新版远程协议），覆盖 MCP 规范定义的全部客户端传输方式。
- **扩充消息模型**：messages 表增加 parts（tool invocation/result/reasoning 等），增加 cost/tokens/model 元数据，使消息可被前端或 API 消费者完整还原。
- **补齐工具集**：增加 edit（oldString/newString diff 编辑）、task（子 agent）、webfetch（URL 抓取）。
- **注入技能到 Engine**：wire 阶段加载技能并拼入 system prompt。
- **实现 per-tool 权限模型**：对齐上游 ask/allow/deny 三态权限。
- **扩展 CLI**：增加 `mcp list`、`session delete`、`models` 等子命令；完善 REPL 为可用的交互式 agent 循环（含工具确认）。

## Capabilities

### New Capabilities

- `react-loop`：ReAct 工具调用循环——消息历史构建、tool_calls 解析、执行、回注、循环终止条件、最大迭代保护。
- `real-providers`：真实 LLM 提供商（OpenAI chat completions + Anthropic messages API），含 API key 配置、流式、tool_use 协议映射。
- `mcp-transports`：MCP 三种传输——stdio（子进程 JSON-RPC over stdin/stdout）、SSE（HTTP GET 长连接 + POST 请求）、Streamable HTTP（单端点 POST + 可选 SSE 响应）；含生命周期管理、超时与错误恢复。
- `message-parts`：结构化消息部件——将 messages 从纯文本 body 扩展为 parts 数组（text/tool-call/tool-result/reasoning），含 cost/tokens/model 元数据。
- `edit-tool`：edit 工具——基于 oldString/newString 的精确文本替换（对齐上游 `src/tool/edit.ts`）。
- `task-tool`：task 工具——启动子 agent（独立会话）执行子任务并返回结果。
- `webfetch-tool`：webfetch 工具——抓取 URL 返回文本/markdown。
- `tool-permissions`：per-tool 权限模型——每个工具可配置 ask/allow/deny，ask 时在 REPL 中交互确认。

### Modified Capabilities

- `agent-runtime`：Engine 从「单次 LLM 调用」重构为「ReAct 循环 + 消息历史」；`CompleteTurn` 签名与行为大幅变更。**BREAKING**
- `llm-and-tools`：Provider 接口扩展为支持 messages 数组 + tools 定义 + tool_choice；不再是单纯的 `Complete(prompt string)`。**BREAKING**
- `persistence`：messages 表结构变更（新增 parts JSON、metadata 列）。**BREAKING**（需迁移）
- `cli-and-config`：新增 `mcp`、`session delete`、`models` 等子命令；REPL 升级为 agent 交互循环。
- `http-api`：complete 端点的请求/响应结构变更以支持 parts 与 metadata。**BREAKING**

## Impact

- **代码**：`internal/runtime/engine.go` 重写；`internal/llm/provider.go` 接口 **BREAKING** 变更；`internal/store` 迁移至 schema v3；新增 `internal/llm/openai.go`、`internal/llm/anthropic.go`、`internal/mcp/stdio.go`、`internal/tool/edit.go` 等。
- **依赖**：引入 `github.com/openai/openai-go`（OpenAI 官方 Go SDK）与 `github.com/anthropics/anthropic-sdk-go`（Anthropic 官方 Go SDK）。
- **API**：`/v1/sessions/{id}/complete` 请求/响应格式变更（需版本协商或文档化 BREAKING）。
- **配置**：新增 `providers`（API key、base URL）、`permissions` 等键。
- **测试**：需为 ReAct 循环、提供商集成（mock HTTP）、MCP stdio 补充集成测试。
