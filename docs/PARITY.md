# 与上游 TypeScript `packages/opencode` 的差异（摘要）

本仓库为 **Go 重新实现**，行为以 `openspec/specs` 与本变更 `openspec/changes/ts-parity-full-impl` 为准。

## 已实现

- **ReAct 循环**：`Engine.CompleteTurn` 实现完整的 ReAct agent 循环 — 接收用户输入、注入系统提示和技能、调用 LLM、解析 tool_calls、执行工具、回注结果、循环直到终止条件（`FinishReason != "tool_calls"` 或达到 `MaxToolRounds`）。
- **真实 LLM 提供商**：通过官方 Go SDK 集成 OpenAI（`openai-go`）和 Anthropic（`anthropic-sdk-go`），支持 Chat 和 ChatStream，完整映射 message/tool_call/tool_result。
- **MCP 传输**：实现三种 MCP JSON-RPC 传输 — stdio（子进程）、SSE（HTTP+SSE 长连接）、Streamable HTTP（单端点 POST）。传输类型可配置或自动推断。
- **新增工具**：`edit`（精确文本替换）、`task`（子 agent 执行）、`webfetch`（URL 抓取）。
- **消息模型 v3**：`parts` 字段支持结构化内容（text/tool_call/tool_result），附带 `model`、`cost_prompt_tokens`、`cost_completion_tokens`、`finish_reason`、`tool_call_id` 元数据。v2→v3 自动迁移。
- **权限模型**：per-tool `allow`/`ask`/`deny` 策略，REPL 模式下 `ask` 会弹出交互确认。
- **插件**：编译期注册（`internal/plugin`），不使用 Go `plugin` 动态库。
- **ACP**：与 HTTP **共端口、共鉴权**；路由 `POST /v1/acp/session/event`。

## 与上游差异

- **配置命名空间**：Go 扩展配置在 `x_opencode_go` 键下，与上游顶层键分离。
- **UI**：上游有 TUI（Terminal UI），Go 版本仅提供 REPL 和 HTTP API。
- **插件系统**：上游使用动态加载，Go 版本使用编译期注册。
