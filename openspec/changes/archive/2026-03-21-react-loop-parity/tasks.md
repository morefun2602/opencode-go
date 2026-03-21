## 1. 指数退避与 retry-after（D2）

- [x] 1.1 新建 `internal/llm/retry.go`，实现 `RetryDelay(attempt int, err error) time.Duration`：base 1s, factor 2, 上限 30s
- [x] 1.2 在 `RetryDelay` 中检查 `err` 是否为 `*RetryableError` 且 `RetryAfter > 0`，若是则使用该值
- [x] 1.3 修改 `internal/llm/errors.go` 的 `Classify`：当检测到 rate limit 相关错误时，尝试从错误字符串中提取 `retry-after` 秒数并包装为 `RetryableError`
- [x] 1.4 编写 `internal/llm/retry_test.go` 单元测试

## 2. 流式路径重试（D1）

- [x] 2.1 在 `internal/runtime/engine.go` 中新增 `streamWithRetry` 方法，封装 `prov.ChatStream` + 重试逻辑
- [x] 2.2 修改 `CompleteTurnStream` 中的 `ChatStream` 调用为 `streamWithRetry`
- [x] 2.3 修改 `callWithRetry` 中的重试等待，使用 `llm.RetryDelay` 替代无等待立即重试
- [x] 2.4 在重试等待前发布 `session.retry` Bus 事件（D10）
- [x] 2.5 重试等待期间检查 `ctx.Err()`，若已取消则立即返回

## 3. Abort/Cancel 传播（D3）

- [x] 3.1 在 `Engine` 中添加 `sessions sync.Map`（sessionID → context.CancelFunc）
- [x] 3.2 在 `CompleteTurn` 和 `CompleteTurnStream` 入口处创建可取消的子 context，存入 sessions map
- [x] 3.3 在循环每轮开头检查 `ctx.Err()`，若非 nil 则跳出循环并持久化已有消息
- [x] 3.4 实现 `Engine.CancelSession(sessionID string)`：查找并调用对应的 cancel 函数，发布 `session.abort` Bus 事件
- [x] 3.5 在循环结束时从 sessions map 中移除 sessionID

## 4. 工具名大小写修复（D4）

- [x] 4.1 修改 `internal/tool/router.go` 的 `Run` 方法：在未找到工具且 name 含大写时，尝试 `strings.ToLower(name)` 再查找一次
- [x] 4.2 编写大小写修复的单元测试

## 5. filterCompacted 消息过滤（D5）

- [x] 5.1 修改 `internal/runtime/engine.go` 的 `loadHistory`：加载消息后从后向前扫描，找到最后一个包含 `[Conversation Summary]` 的 user 消息作为 compaction 点
- [x] 5.2 仅保留 compaction 点及之后的消息
- [x] 5.3 编写 filterCompacted 的单元测试

## 6. maybeCompact 修复（D6）

- [x] 6.1 修改 `maybeCompact`：当消息数超过阈值时，异步调用 `Compaction.Process` 并将结果保存
- [x] 6.2 添加错误日志记录
- [x] 6.3 编写 maybeCompact 的单元测试

## 7. MAX_STEPS 警告注入（D7）

- [x] 7.1 定义 `maxStepsWarning` 常量：提示模型步数即将达到上限，应总结当前进度并结束
- [x] 7.2 在循环中检测 `round == maxRounds-1`，追加警告消息到 msgs，将 tdefs 设为空
- [x] 7.3 编写 MAX_STEPS 注入的单元测试

## 8. Noop 工具注入（D8）

- [x] 8.1 在 `CompleteTurn`/`CompleteTurnStream` 中，当 `tdefs` 为空且 `msgs` 中存在 tool_call Part 时，注入 `_noop` 工具定义
- [x] 8.2 在 `tool.Router` 或内置注册中注册 `_noop` 工具实现（返回 "noop"）
- [x] 8.3 编写 noop 注入的单元测试

## 9. 结构化输出（D9）

- [x] 9.1 在 `Engine` 中检查 `StructuredOutputSchema` 是否非空
- [x] 9.2 当工具调用循环完成后的最后一轮，追加 `_structured_output` 工具（schema 来自配置）
- [x] 9.3 将模型通过该工具返回的结果解析为结构化输出
- [x] 9.4 编写结构化输出的单元测试

## 10. 集成验证

- [x] 10.1 运行 `go build ./...` 确保编译通过
- [x] 10.2 运行 `go test ./...` 确保全部测试通过
- [x] 10.3 运行 `go vet ./...` 检查代码质量
- [x] 10.4 验证流式重试：模拟 RateLimit 错误确认重试行为
- [x] 10.5 验证 cancel 传播：调用 CancelSession 确认循环终止
- [x] 10.6 验证 MAX_STEPS：设置 MaxToolRounds=2 确认警告注入
