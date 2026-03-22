## Context

Go 版本（`opencode-go`）已有骨架包：`internal/runtime`、`internal/llm`、`internal/tool`、`internal/mcp`、`internal/store` 等。但 `Engine.CompleteTurn` 直接将用户文本传给 `LLM.Complete(string)`——没有消息历史、没有工具定义注入、没有 tool_calls 解析与执行循环。LLM 层仅有 stub，MCP 仅有 `NullTransport`，消息模型为纯文本 `(role, body)`。

上游 TS `packages/opencode` 采用：

- **AI SDK** + 多提供商（OpenAI、Anthropic、Google 等），返回结构化 `ToolInvocation` / `ToolResult`
- **消息 v2**：parts 数组（text/tool-call/tool-result/reasoning/snapshot 等），含 cost/tokens/model 元数据
- **ReAct 循环**：会话级消息历史 → LLM → 解析 tool_calls → 执行 → 回注 → 循环直到无 tool_calls 或达到上限
- **MCP**：stdio / HTTP / SSE 传输 + OAuth
- **权限**：per-tool ask/allow/deny

本设计描述如何在 Go 侧实现等价行为。

## Goals / Non-Goals

**Goals:**

- 定义新 `Provider` 接口以支持 messages 数组 + tools 定义 + 结构化响应（tool_calls / text / usage）
- 基于官方 Go SDK 实现 OpenAI 与 Anthropic 提供商（`github.com/openai/openai-go` + `github.com/anthropics/anthropic-sdk-go`）
- 在 `runtime.Engine` 中实现 ReAct 循环：消息历史加载 → system prompt（含技能）→ tools schema → LLM → 解析 → 执行 → 回注 → 终止判断
- 持久化扩展为结构化 parts + metadata
- 实现 stdio MCP Transport
- 实现 edit / task / webfetch 工具
- 实现 per-tool 权限
- REPL 升级为真正的 agent 循环（含工具确认）

**Non-Goals:**

- 复刻上游完整 TUI（`@opentui/solid`）——REPL 足够，TUI 留作独立变更
- 实现 Google / Bedrock / Azure 等提供商——仅 OpenAI + Anthropic
- 实现 MCP OAuth——鉴权留作后续变更
- 实现 LSP / snapshot / share / 桌面端相关功能
- 实现 batch tool / websearch / codesearch（可在后续变更中补齐）

## Decisions

### 1. Provider 接口重构（BREAKING）

**决策**：将 `Provider` 从 `Complete(ctx, prompt string)` 重构为：

```go
type Message struct {
    Role    string // "system" | "user" | "assistant" | "tool"
    Content string // 纯文本（user/system/assistant 文本部分）
    Parts   []Part // 结构化部件（tool_call / tool_result 等）
}

type Part struct {
    Type       string         // "text" | "tool_call" | "tool_result"
    Text       string         // type=text 时
    ToolCallID string         // type=tool_call / tool_result 时
    ToolName   string         // type=tool_call 时
    Args       map[string]any // type=tool_call 时
    Result     string         // type=tool_result 时
    IsError    bool           // type=tool_result 时
}

type ToolDef struct {
    Name        string
    Description string
    Parameters  map[string]any // JSON Schema
}

type Usage struct {
    PromptTokens     int
    CompletionTokens int
}

type Response struct {
    Message  Message
    Usage    Usage
    Model    string
    FinishReason string // "stop" | "tool_calls" | "length"
}

type Provider interface {
    Name() string
    Chat(ctx context.Context, msgs []Message, tools []ToolDef) (*Response, error)
    ChatStream(ctx context.Context, msgs []Message, tools []ToolDef, chunk func(*Response) error) (*Response, error)
}
```

- **理由**：上游 AI SDK 的核心抽象就是 `messages + tools → response`。Go 侧如果沿用 `Complete(string)` 无法支持工具调用。`Message.Parts` 同时用于请求（tool_result 回注）和响应（tool_call 输出），与 OpenAI / Anthropic 的 wire format 均可直接映射。
- **备选**：继续用 `Complete(string)` + 在 Engine 层手动拼 JSON — **否决**（职责混淆、测试困难）。

### 2. ReAct 循环（`runtime.Engine`）

**决策**：`Engine.CompleteTurn` 重写为以下流程：

1. 从 Store 加载会话历史（`ListMessages`），转为 `[]llm.Message`
2. 构建 system prompt：基础指令 + `skill.InjectPrompt`
3. 追加当前 user message
4. 收集可用工具定义（内置 + MCP）为 `[]llm.ToolDef`
5. **循环**（最大 `MaxToolRounds` 次，默认 25）：
   a. 调用 `Provider.Chat(ctx, messages, tools)`
   b. 将 assistant response 追加到 messages
   c. 如果 `FinishReason != "tool_calls"`：结束循环
   d. 遍历 response 中的 tool_calls：
      - 检查权限（`tool-permissions`）→ 若 `ask` 且未确认：暂停或拒绝
      - 通过 `tool.Router.Run` 执行
      - 构建 tool_result message 追加到 messages
   e. 下一轮循环
6. 持久化全部新 messages（含中间 tool_call/result）

- **理由**：这是 ReAct 的标准模式，上游 TS 侧亦如此。将循环放在 Engine 而非 Provider 中，使 Provider 只负责单次 HTTP 调用。
- **备选**：Provider 内部做循环 — **否决**（Provider 不应知道工具执行细节）。
- **`MaxToolRounds`**：可配置（`max_tool_rounds`），默认 25（与上游一致），防止无限循环。

### 3. 消息持久化（schema v3）

**决策**：升级 `messages` 表：

```sql
ALTER TABLE messages ADD COLUMN parts TEXT NOT NULL DEFAULT '[]';    -- JSON array of Part
ALTER TABLE messages ADD COLUMN model TEXT NOT NULL DEFAULT '';
ALTER TABLE messages ADD COLUMN cost_prompt_tokens INTEGER NOT NULL DEFAULT 0;
ALTER TABLE messages ADD COLUMN cost_completion_tokens INTEGER NOT NULL DEFAULT 0;
ALTER TABLE messages ADD COLUMN finish_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE messages ADD COLUMN tool_call_id TEXT NOT NULL DEFAULT '';
```

- `parts`：JSON 序列化 `[]Part`，存储结构化消息部件
- `model` / `cost_*` / `finish_reason`：LLM 响应元数据
- `tool_call_id`：role=tool 的消息关联对应的 tool_call
- 旧数据（schema v2）的 `body` 字段继续保留作为纯文本后备；迁移脚本将旧 body 包装为 `[{"type":"text","text":"..."}]` 写入 parts

- **理由**：上游消息 v2 以 parts 为核心；Go 侧若仍用纯 body 字符串，无法还原 tool_call/result 结构。JSON 列在 SQLite 中足够（无需关系化 parts 表）。
- **备选**：分出 `message_parts` 关系表 — **否决**（增加查询复杂度，SQLite 单文件场景下 JSON 列更简单）。

### 4. OpenAI 提供商（官方 Go SDK）

**决策**：`internal/llm/openai.go`，基于官方 SDK `github.com/openai/openai-go`。

- 使用 `openai.NewClient()` 创建客户端，SDK 自动从 `OPENAI_API_KEY` 环境变量读取 key
- 通过 `option.WithBaseURL()` 支持自定义 base URL（兼容 Azure OpenAI / 私有部署）
- 将 `[]llm.Message` 映射到 SDK 的 `openai.ChatCompletionMessageParamUnion`（含 tool_calls 与 role=tool 等）
- 将 `[]llm.ToolDef` 映射到 SDK 的 `openai.ChatCompletionToolParam`
- 非流式调用 `client.Chat.Completions.New()`；流式调用 `client.Chat.Completions.NewStreaming()`
- SDK 自行处理 SSE 解析、HTTP 重试、错误类型化，省去手动拼接逻辑
- API key 优先级：`providers.openai.api_key` 配置 > `OPENAI_API_KEY` 环境变量（SDK 默认行为）

- **理由**：`openai/openai-go` 是 OpenAI 官方维护的 Go SDK，类型安全、wire format 自动跟进 API 变更，减少自行维护序列化/反序列化的负担。
- **备选**：直接 `net/http` 调用 — **否决**（需自行处理 SSE 解析、流式 tool_calls delta 拼接、错误体解析等，工作量与维护成本显著高于引入 SDK）。

### 5. Anthropic 提供商（官方 Go SDK）

**决策**：`internal/llm/anthropic.go`，基于官方 SDK `github.com/anthropics/anthropic-sdk-go`。

- 使用 `anthropic.NewClient()` 创建客户端，SDK 自动从 `ANTHROPIC_API_KEY` 环境变量读取 key
- Anthropic 的 tool_use 与 OpenAI 格式不同（`tool_use` content block + `tool_result` content block），SDK 提供类型化的 `anthropic.ContentBlockParam` 进行映射
- 非流式调用 `client.Messages.New()`；流式调用 `client.Messages.NewStreaming()`
- API key 优先级：`providers.anthropic.api_key` 配置 > `ANTHROPIC_API_KEY` 环境变量

- **理由**：`anthropics/anthropic-sdk-go` 是 Anthropic 官方维护的 Go SDK，与 `openai-go` 设计风格一致（均基于 `stainless` 生成），tool_use 的复杂映射由 SDK 类型系统保证正确性。
- **备选**：直接 `net/http` — **否决**（Anthropic tool_use 的 content block 结构比 OpenAI 更复杂，手动解析容易出错且难以跟进 API 演进）。

### 6. MCP Transport（stdio + SSE + Streamable HTTP）

**决策**：实现三种 MCP 传输，覆盖 MCP 规范定义的全部客户端传输方式。

#### 6a. stdio Transport — `internal/mcp/stdio.go`

- 通过 `exec.Command` 启动子进程，stdin/stdout 上跑 JSON-RPC 2.0
- 写入带换行的 JSON 请求到 stdin，从 stdout 逐行读取响应
- 子进程生命周期由 `Client` 管理：`Connect` 时启动，`Close` 时发 SIGTERM + 等待
- 超时通过 `context.Context` 传递
- stderr 转发到日志
- 适用场景：本地 MCP 服务端（如 filesystem、sqlite 等社区工具）

#### 6b. SSE Transport — `internal/mcp/sse.go`

- 对应 MCP 规范中的 **HTTP with SSE** 传输（旧版远程协议）
- 客户端先 GET 服务端的 SSE 端点（如 `/sse`），建立长连接接收服务端事件
- 服务端在 SSE 流中发送一个 `endpoint` 事件，包含后续 JSON-RPC 请求要 POST 到的 URL
- 客户端将 JSON-RPC 请求 POST 到该 URL，响应通过 SSE 流异步返回（按 `id` 关联）
- 连接断开时自动重连（可配置最大重试次数）
- 适用场景：仅支持旧版 SSE 协议的远程 MCP 服务端

#### 6c. Streamable HTTP Transport — `internal/mcp/streamable.go`

- 对应 MCP 规范中的 **Streamable HTTP** 传输（2025-03 新增，替代旧版 SSE）
- 客户端将 JSON-RPC 请求 POST 到单一端点（如 `/mcp`）
- 服务端可选择返回：
  - 普通 JSON 响应（`application/json`）——适用于无流式需求的简单请求
  - SSE 流（`text/event-stream`）——适用于流式结果或服务端主动推送
- 支持可选的 `Mcp-Session-Id` header 实现有状态会话
- 支持服务端通过 GET 端点发起 SSE 推送通知（服务端 → 客户端方向）
- 适用场景：新一代远程 MCP 服务端（推荐的远程传输方式）

#### 传输选择策略

配置中每个 MCP 服务端条目新增 `transport` 字段：

```json
{
  "mcp_servers": [
    { "name": "local-fs", "transport": "stdio", "command": "mcp-fs", "args": ["--root", "."] },
    { "name": "remote-legacy", "transport": "sse", "url": "https://example.com/sse" },
    { "name": "remote-new", "transport": "streamable_http", "url": "https://example.com/mcp" }
  ]
}
```

- `transport` 缺省值为 `"stdio"`（向后兼容现有配置）
- 有 `command` 字段时自动推断为 stdio；有 `url` 字段时自动推断为 streamable_http（优先新协议）
- 三种传输均实现同一 `Transport` 接口，`Client` 无需感知传输细节

- **理由**：上游 TS SDK 已支持全部三种传输（`StdioClientTransport`、`SSEClientTransport`、`StreamableHTTPClientTransport`）。stdio 覆盖本地场景；Streamable HTTP 是 MCP 规范推荐的远程传输；SSE 保证与旧版远程服务端兼容。
- **备选**：仅实现 stdio — **否决**（无法连接任何远程 MCP 服务端，限制了实际可用性）。

### 7. edit 工具

**决策**：`internal/tool/edit.go`，参数为 `path`、`old_string`、`new_string`。

- 读取文件 → `strings.Replace(content, oldString, newString, 1)` → 写回
- 若 oldString 不存在则报错
- 若 oldString 出现多次则报错（要求唯一匹配，与上游一致）
- 路径受 `ResolveUnder` 限制

### 8. task 工具（子 agent）

**决策**：`internal/tool/task.go`。

- 创建临时子会话（新 sessionID），复用同一 Engine + Store
- 运行 `Engine.CompleteTurn(ctx, workspaceID, subSessionID, taskPrompt)`
- 返回子 agent 的最终文本输出
- 子会话可配置最大深度（防嵌套爆炸），默认 2

### 9. webfetch 工具

**决策**：`internal/tool/webfetch.go`，参数为 `url`。

- `net/http` GET → 读取 body → 截断到 `MaxOutputBytes` → 返回纯文本
- 可选尝试用 `html.Parse` 提取 `<body>` 文本（简易去标签）
- 超时与 `bash` 工具一致（`BashTimeoutSec`）

### 10. 权限模型

**决策**：扩展 `internal/policy` 为 per-tool 权限。

- 配置 `permissions`：`map[string]string`（工具名 → `"ask"` / `"allow"` / `"deny"`），默认全部 `"allow"`
- `ask` 模式下，Engine 在执行 tool_call 前调用 `Confirm` 回调（由 REPL 注入）
- `deny` 直接返回错误

### 11. HTTP API 变更

**决策**：`complete` 端点响应格式变更：

- 非流式响应从 `{"reply":"..."}` 变为 `{"messages":[...]}` — 返回本轮新增的所有消息（含中间 tool_call/result）
- 流式 SSE 事件从纯 `data: <chunk>` 变为 `data: {"type":"text","text":"..."}` / `data: {"type":"tool_call",...}` 等结构化事件
- 保持路径 `/v1/sessions/{id}/complete` 不变；旧格式不再支持（**BREAKING**）

### 12. 配置扩展

**决策**：使用顶层配置新增：

```json
{
  "providers": {
    "openai": { "api_key": "", "base_url": "", "model": "gpt-4o" },
    "anthropic": { "api_key": "", "model": "claude-sonnet-4-20250514" }
  },
  "default_provider": "openai",
  "default_model": "",
  "max_tool_rounds": 25,
  "permissions": { "bash": "ask", "write": "ask" },
  "skills_dir": ""
}
```

环境变量 `OPENAI_API_KEY`、`ANTHROPIC_API_KEY` 可覆盖配置文件中的 key。

## Risks / Trade-offs

- **[Risk]** Provider 接口 BREAKING 导致所有测试与 wire 需更新 → **Mitigation**：stub 同步实现新接口；所有测试在本变更中修复。
- **[Risk]** OpenAI / Anthropic API 变更导致 SDK 类型不兼容 → **Mitigation**：使用官方 SDK 的 semver 版本锁定；SDK 由各厂商与 API 同步更新，跟进成本低于自维护 HTTP 客户端。
- **[Risk]** stdio MCP 子进程泄漏 → **Mitigation**：Context 取消时 Kill 子进程；`Client.Close` 强制清理。
- **[Risk]** ReAct 无限循环 → **Mitigation**：`MaxToolRounds` 硬限（默认 25）；每轮超时由 `context.Context` 控制。
- **[Trade-off]** 消息 parts 用 JSON 列 vs 关系表 → JSON 更简单，但不支持高效的部件级查询。考虑到 SQLite 单文件场景且查询需求为整条消息加载，JSON 列足够。
- **[Trade-off]** 仅实现 OpenAI + Anthropic 两个提供商 → 覆盖了最常用场景；其他提供商（Google、Bedrock 等）结构类似，后续可快速扩展。

## Open Questions

- OpenAI `tool_choice` 策略（`auto` / `required` / `none`）是否暴露为配置或由 Engine 自动决定？初步倾向 Engine 自动（首轮 `auto`，后续 `auto`）。
- 流式 tool_calls 的拼接策略——OpenAI SDK 的 `ChatCompletionStreamResponse` 已在 accumulator 中拼接 tool_calls delta，Provider 层直接使用 SDK 的 `AccumulateChunk` 即可；Anthropic SDK 同理提供事件级 API，无需手动拼接。
