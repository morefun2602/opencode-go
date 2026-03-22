# 与上游 TypeScript `packages/opencode` 的差异（摘要）

本仓库为 **Go 重新实现**，行为以 `openspec/specs` 与本变更 `openspec/changes/ts-parity-full-impl` 为准。

## 已实现

- **ReAct 循环**：`Engine.CompleteTurn` 实现完整的 ReAct agent 循环 — 接收用户输入、注入系统提示和技能、调用 LLM、解析 tool_calls、执行工具、回注结果、循环直到终止条件（`FinishReason != "tool_calls"` 或达到 `MaxToolRounds`）。
- **真实 LLM 提供商**：通过官方 Go SDK 集成 OpenAI（`openai-go`）和 Anthropic（`anthropic-sdk-go`），支持 Chat 和 ChatStream，完整映射 message/tool_call/tool_result。
- **MCP 传输**：实现三种 MCP JSON-RPC 传输 — stdio（子进程）、SSE（HTTP+SSE 长连接）、Streamable HTTP（单端点 POST）。传输类型可配置或自动推断。
- **MCP 资源能力**：连接阶段拉取资源清单，并支持 `ListResources` / `ReadResource`。
- **MCP OAuth 接入**：支持 OAuth token 文件存储、请求头注入、401 场景失效后重试认证。
- **工具执行入口治理**：`Registry.Run` 增加统一 schema 校验与标准化校验错误。
- **工具能力补齐**：新增 `todoread`；支持工作区 `.opencode/tool/*.json` 与 `.opencode/tools/*.json` 的自定义工具加载。
- **模型路由策略**：按模型动态裁剪 `apply_patch` 与 `edit/write/multiedit` 暴露策略。
- **Agent/ReAct 语义增强**：`ask` 在无 Confirm handler 时默认拒绝；Doom loop 支持配置窗口（`doom_loop_window`）并接入审计。
- **ReAct 可观测性**：发布 `react.round.start/finish`、`react.blocked`、`react.compact.*`、`session.summary` 事件。
- **Summary 生命周期**：step 结束后接入 SessionSummary 增量摘要链路。
- **消息模型 v3**：`parts` 字段支持结构化内容（text/tool_call/tool_result），附带 `model`、`cost_prompt_tokens`、`cost_completion_tokens`、`finish_reason`、`tool_call_id` 元数据。v2→v3 自动迁移。
- **权限模型**：per-tool `allow`/`ask`/`deny` 策略，REPL 模式下 `ask` 会弹出交互确认。
- **插件**：编译期注册（`internal/plugin`），不使用 Go `plugin` 动态库。
- **ACP**：与 HTTP **共端口、共鉴权**；路由 `POST /v1/acp/session/event`。

## 与上游差异

- **配置形态**：Go 版当前以工程内约定配置结构为主，与上游配置项存在映射差异（例如 MCP OAuth 字段与自定义工具加载协议）。
- **UI**：上游有 TUI（Terminal UI），Go 版本仅提供 REPL 和 HTTP API。
- **插件系统**：上游支持动态加载 JS/TS 插件；Go 版本主路径仍为编译期注册，自定义工具通过 JSON + shell 命令协议补齐。
- **MCP 高阶能力**：与上游相比，prompts / tools-changed 事件驱动刷新仍有差距（当前以连接快照 + 调用时恢复为主）。
