## 1. 依赖与类型基础

- [x] 1.1 引入 `github.com/openai/openai-go` 与 `github.com/anthropics/anthropic-sdk-go` 依赖（`go get`）
- [x] 1.2 重构 `internal/llm/provider.go`：定义 `Message`、`Part`、`ToolDef`、`Usage`、`Response` 类型与新 `Provider` 接口（`Chat` + `ChatStream`），删除旧 `Complete` 方法
- [x] 1.3 更新 `internal/llm/stub.go`：实现新 `Provider` 接口（`Chat` 返回 echo 响应，`ChatStream` 直接回调后返回）
- [x] 1.4 更新 `internal/llm/errors.go`：确保 `Kind` 分类覆盖 SDK 返回的错误类型（Timeout、RateLimit、Auth）

## 2. 消息持久化（schema v3）

- [x] 2.1 更新 `internal/store/store.go`：`MessageRow` 新增 `Parts`、`Model`、`CostPromptTokens`、`CostCompletionTokens`、`FinishReason`、`ToolCallID` 字段
- [x] 2.2 更新 `internal/store/migrate.go`：`schemaVersion` 升至 3；实现 v2→v3 迁移（ALTER TABLE 添加列 + 旧 body 包装为 parts JSON）
- [x] 2.3 更新 `internal/store/sqlite.go`：`AppendTurn` 写入新列；`ListMessages` 读取新列并反序列化 parts

## 3. LLM 提供商

- [x] 3.1 实现 `internal/llm/openai.go`：基于 `openai-go` SDK，实现 `Chat`（`client.Chat.Completions.New`）和 `ChatStream`（`NewStreaming` + accumulator），完成 Message/ToolDef 映射
- [x] 3.2 实现 `internal/llm/anthropic.go`：基于 `anthropic-sdk-go` SDK，实现 `Chat`（`client.Messages.New`）和 `ChatStream`（`NewStreaming`），完成 tool_use content block 到 Part 的映射
- [x] 3.3 实现提供商注册工厂函数：根据配置 `default_provider` 选择实例化 OpenAI / Anthropic / stub

## 4. MCP 传输

- [x] 4.1 实现 `internal/mcp/stdio.go`：`StdioTransport` 结构体，`exec.Command` 启动子进程，stdin 写 JSON-RPC、stdout 逐行读响应，生命周期管理（SIGTERM + SIGKILL）
- [x] 4.2 实现 `internal/mcp/sse.go`：`SSETransport` 结构体，GET 建立 SSE 长连接，等待 endpoint 事件获取 POST URL，POST 发送 JSON-RPC，SSE 流异步接收响应，断线重连
- [x] 4.3 实现 `internal/mcp/streamable.go`：`StreamableHTTPTransport` 结构体，POST 到单端点，解析 `application/json` 或 `text/event-stream` 响应，`Mcp-Session-Id` header 管理
- [x] 4.4 更新 `internal/mcp/transport.go`：删除 `NullTransport`，确保 `Transport` 接口涵盖三种实现所需方法
- [x] 4.5 更新 `internal/mcp/client.go`：`Connect` 按传输类型初始化，传输选择逻辑（transport 字段 / 自动推断）

## 5. 工具实现

- [x] 5.1 实现 `internal/tool/edit.go`：`edit` 工具——读文件、`strings.Count` 校验唯一、`strings.Replace` 写回，路径受 `ResolveUnder` 限制
- [x] 5.2 实现 `internal/tool/task.go`：`task` 工具——创建子会话 ID、复用 Engine 调用 `CompleteTurn`、嵌套深度检查、返回最终文本
- [x] 5.3 实现 `internal/tool/webfetch.go`：`webfetch` 工具——HTTP GET、`MaxOutputBytes` 截断、简易 HTML 去标签、超时控制
- [x] 5.4 更新 `internal/tool/builtin.go`：将 `edit`、`task`、`webfetch` 注册到内置工具列表，定义各工具的 JSON Schema 参数

## 6. 权限模型

- [x] 6.1 更新 `internal/policy/policy.go`：添加 `Permissions map[string]string` 字段与 `CheckPermission(toolName) string` 方法
- [x] 6.2 在 `runtime.Engine` 中添加 `Confirm func(name string, args map[string]any) (bool, error)` 回调字段
- [x] 6.3 实现权限检查流程：Engine 在执行 tool_call 前查询 `CheckPermission`，deny 返回拒绝 tool_result，ask 调用 Confirm（未注入则降级 allow）

## 7. ReAct 循环

- [x] 7.1 重写 `internal/runtime/engine.go` 的 `CompleteTurn`：加载历史消息 → 构建 system prompt（含技能注入）→ 追加 user message → 收集工具定义 → 循环调用 Provider
- [x] 7.2 实现 tool_calls 解析与执行循环：遍历 response.Parts 中 type=tool_call 的部件，通过 Router.Run 执行，构建 tool_result 消息回注
- [x] 7.3 实现循环终止条件：FinishReason != "tool_calls" 或达到 MaxToolRounds 上限
- [x] 7.4 实现 LLM 重试逻辑：对 RateLimit/Timeout 错误自动重试（最大 LLMMaxRetries 次），Auth 错误直接返回
- [x] 7.5 实现消息批量持久化：循环结束后将全部新增消息在单一事务中写入 Store
- [x] 7.6 实现压缩检查：轮次结束后检查消息历史长度是否超过 CompactionTurns，超过时记录日志
- [x] 7.7 重写 `CompleteTurnStream`：与 `CompleteTurn` 共享循环逻辑，使用 `ChatStream` 替代 `Chat`

## 8. 配置扩展

- [x] 8.1 更新 `internal/config/config.go`：`File.Go` 新增 `Providers`（含 OpenAI/Anthropic 的 api_key/base_url/model）、`DefaultProvider`、`DefaultModel`、`MaxToolRounds`、`Permissions`、`SkillsDir`
- [x] 8.2 更新 `MCPServerFile` 结构体：添加 `Transport`、`URL` 字段
- [x] 8.3 更新 `Defaults()`、`merge()`、`mergeFlags()`：处理新增字段的默认值与合并逻辑
- [x] 8.4 实现环境变量覆盖：`OPENAI_API_KEY` / `ANTHROPIC_API_KEY` 覆盖配置文件 api_key

## 9. HTTP API 变更

- [x] 9.1 更新 `internal/server/handlers.go`：`complete` 端点非流式响应改为 `{"messages":[...]}` 格式
- [x] 9.2 更新流式 SSE 事件格式：每个 data 行改为 `{"type":"text",...}` / `{"type":"tool_call",...}` / `{"type":"tool_result",...}` 结构化 JSON
- [x] 9.3 更新 `internal/server/openapi.go`：OpenAPI 规范文档反映新的请求/响应格式

## 10. CLI 与 REPL

- [x] 10.1 更新 `internal/cli/repl.go`：从简单 stdin 循环升级为 agent 循环——注入 Confirm 回调，打印工具调用详情，等待用户 y/n 确认
- [x] 10.2 更新 `internal/cli/wire.go`：根据新配置初始化真实 Provider（OpenAI/Anthropic/stub）、构建 MCP 客户端（按 transport 类型选择传输）、注入 Permissions
- [x] 10.3 确保 `sessions list`、`tools list`、`skills list` 子命令与新数据模型兼容

## 11. 测试

- [x] 11.1 更新 `internal/tool/router_test.go`：适配新 Provider 接口
- [x] 11.2 编写 `internal/store/migrate_from_v2_test.go`：测试 v2→v3 迁移（新列存在、body→parts 转换正确）
- [x] 11.3 编写 `internal/llm/openai_test.go`：使用 httptest 模拟 OpenAI API，测试 Chat/ChatStream 映射逻辑
- [x] 11.4 编写 `internal/llm/anthropic_test.go`：使用 httptest 模拟 Anthropic API，测试 tool_use 映射
- [x] 11.5 编写 `internal/mcp/stdio_test.go`：使用 echo 脚本模拟 MCP 服务端，测试 JSON-RPC 通信与进程生命周期
- [x] 11.6 编写 `internal/runtime/engine_test.go`：使用 stub Provider，测试 ReAct 循环（含 tool_calls → 执行 → 回注 → 终止）
- [x] 11.7 编写 `internal/tool/edit_test.go`：测试唯一匹配、不存在、多次匹配、越界路径四种场景
- [x] 11.8 更新 `internal/server/list_test.go`：适配新 complete 响应格式与结构化 SSE 事件

## 12. 文档更新

- [x] 12.1 更新 `docs/CONFIG.md`：新增 providers、permissions、max_tool_rounds、mcp_servers（含 transport/url）等配置键说明
- [x] 12.2 更新 `docs/HTTP.md`：complete 端点新响应格式、结构化 SSE 事件格式
- [x] 12.3 更新 `docs/PARITY.md`：标记 ReAct 循环、真实提供商、MCP transports、edit/task/webfetch 工具已实现
- [x] 12.4 更新 `docs/PERSISTENCE.md`：messages 表 schema v3 列定义
