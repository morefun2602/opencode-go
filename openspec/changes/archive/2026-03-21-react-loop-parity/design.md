## Decisions

### D1: 流式重试 — 提取通用 retryLoop

**选择**：将重试逻辑从 `callWithRetry` 提取为通用的 `retryLoop` 函数，支持任意 `func() error` 回调。`CompleteTurnStream` 中的 `ChatStream` 调用通过同一 `retryLoop` 获得 Timeout/RateLimit 重试能力。

**理由**：当前 `callWithRetry` 仅封装 `Chat()`，流式路径直接调用 `ChatStream()` 没有重试。TS 版本的 `processor.ts` 对流式调用同样有完整的重试逻辑。提取通用函数避免代码重复。

**影响**：`internal/runtime/engine.go`，新增 `streamWithRetry` 方法。

---

### D2: 指数退避 — 新增 internal/llm/retry.go

**选择**：在 `internal/llm/` 新建 `retry.go`，实现 `RetryDelay(attempt int, err error) time.Duration`。策略为指数退避（base 1s, factor 2, max 30s），若错误包含 `RetryableError` 且 `RetryAfter > 0` 则使用该值。`Classify()` 增强为在检测到 `retry-after` 时返回 `RetryableError`。

**理由**：当前重试是无等待的立即重试，对 rate limit 场景无效。TS 版本实现了指数退避并解析 `retry-after` 头。Go 的 SDK 错误类型中可能携带 retry 信息。

**影响**：`internal/llm/retry.go`（新建）、`internal/llm/errors.go`（增强 Classify）、`internal/runtime/engine.go`（使用 RetryDelay）。

---

### D3: Abort/Cancel 传播 — 使用 context.Context

**选择**：利用 Go 原生的 `context.Context` 取消机制。在 Engine 上增加 `CancelFunc` 映射（sessionID → cancel），外部通过 `Engine.CancelSession(sessionID)` 触发。循环每轮开头检查 `ctx.Err()`，LLM 调用和工具执行已接受 ctx 参数。新增 `session.abort` Bus 事件。

**理由**：Go 的 context 机制天然适合取消传播，无需引入新的 Abort 抽象。TS 用 AbortController 是 JS 的等价物。

**影响**：`internal/runtime/engine.go`（增加 CancelSession、循环检查 ctx.Err）。

---

### D4: 工具名大小写修复 — tool.Router 增强

**选择**：在 `tool.Router.Run()` 中，当 name 完全未找到时，先尝试 `strings.ToLower(name)` 匹配。匹配成功则使用小写名执行。匹配失败仍走现有 invalid 路由。

**理由**：TS 版本的 `experimental_repairToolCall` 实现了此逻辑。部分模型会返回不精确的工具名大小写，修复可减少无效调用。

**影响**：`internal/tool/router.go`。

---

### D5: filterCompacted 消息过滤 — loadHistory 增强

**选择**：在 `loadHistory` 中，从后向前扫描消息行。当遇到 `role=user` 且 content 包含 `[Conversation Summary]` 标记时，仅保留该消息及其后所有消息。这模拟 TS 的 `filterCompacted` 行为。

**理由**：当前 `loadHistory` 加载全部 100k 消息，compaction 后旧消息仍然存在会导致 token 浪费和潜在的重复上下文。TS 版本明确过滤压缩前的消息。

**影响**：`internal/runtime/engine.go` 的 `loadHistory`。

---

### D6: maybeCompact 修复 — 实际执行压缩

**选择**：将 `maybeCompact` 从仅日志改为：当消息数超过 `CompactionTurns*2` 时，调用 `Compaction.Process()` 执行压缩并持久化结果。使用 goroutine 异步执行以不阻塞主返回。

**理由**：当前实现是残留的占位代码，仅打日志。TS 中的等价逻辑会实际执行压缩。

**影响**：`internal/runtime/engine.go` 的 `maybeCompact`。

---

### D7: MAX_STEPS 警告注入 — 最后一轮禁用工具

**选择**：在循环的最后一轮（`round == maxRounds-1`）时，向消息列表末尾追加 MAX_STEPS 提示消息，告知模型即将达到步数限制，应结束当前工作。同时将工具列表设为空，迫使模型生成文本回复而非工具调用。

**理由**：TS 版本在最后一步注入 `MAX_STEPS` 消息并禁用工具。Go 版本静默截断导致模型可能在工具调用中间中断，用户体验差。

**影响**：`internal/runtime/engine.go` 循环内部。

---

### D8: Noop 工具注入 — 历史兼容

**选择**：在 `collectTools()` 返回空列表但消息历史中存在 `tool_call` Part 时，注入一个名为 `_noop` 的占位工具（参数为空对象，执行返回 "noop"）。

**理由**：部分 LLM Provider（如 OpenAI）在历史中有 tool_calls 但当前无工具定义时会报错。TS 版本添加 `_noop` 解决此问题。

**影响**：`internal/runtime/engine.go` 的工具收集逻辑。

---

### D9: 结构化输出 — Engine.StructuredOutput 模式

**选择**：当 `Engine.StructuredOutputSchema` 非空时，最后一轮 LLM 调用使用结构化输出模式。在 `ToolDef` 列表中附加 `_structured_output` 工具（schema 来自配置），模型的 tool_call 结果即为结构化输出。或者在 Provider 级别支持 `response_format` 参数。

**选择简化方案**：鉴于 Go 版 Provider 接口已稳定，采用在最后一轮追加特殊工具的方式实现，避免修改 Provider 接口。

**影响**：`internal/runtime/engine.go`。

---

### D10: 重试状态回馈 — Bus 事件

**选择**：在 `callWithRetry` / `streamWithRetry` 中，每次进入重试时发布 `session.retry` Bus 事件，payload 包含 `{session_id, attempt, delay_ms, error}`。

**理由**：TS 版本通过 `SessionStatus.set` 设置 retry 状态。Go 版本使用 Bus 事件机制更符合其架构。

**影响**：`internal/runtime/engine.go`。

## File Changes

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `internal/llm/retry.go` | 新建 | RetryDelay 指数退避 + retry-after 解析 |
| `internal/llm/errors.go` | 修改 | Classify 增强，提取 RetryAfter |
| `internal/runtime/engine.go` | 修改 | 主循环增强（流式重试、abort、filterCompacted、maybeCompact、MAX_STEPS、noop、结构化输出、重试事件） |
| `internal/tool/router.go` | 修改 | 工具名大小写修复 |
| `internal/runtime/engine_test.go` | 修改 | 新增测试场景 |
| `internal/llm/retry_test.go` | 新建 | 重试延迟测试 |
