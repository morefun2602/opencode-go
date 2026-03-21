## Why

深入对比 TypeScript 和 Go 版本的 ReAct 循环实现后，发现 Go 版本在核心可靠性和正确性方面存在多项关键缺失。流式路径缺少超时/限流重试、无取消传播机制、Compaction 后的消息过滤缺失、`maybeCompact` 仅记日志不执行、最大步数到达时无警告注入、历史含 tool_calls 但当前无工具时缺少兼容处理。这些差距直接影响框架在企业级场景下的稳定性和正确性。

## What Changes

### 核心可靠性

- 为 `CompleteTurnStream` 添加 Timeout/RateLimit 重试，与非流式路径对齐
- 实现指数退避重试策略，支持 `retry-after` 响应头解析
- 添加 abort/cancel 传播机制，支持通过 context 或 `AbortFunc` 中断循环和 LLM 调用
- 实现工具名大小写修复（case-insensitive tool name repair），未知工具先尝试小写匹配

### 正确性

- 实现 `filterCompacted` 消息过滤：Compaction 完成后仅加载压缩点之后的消息，避免重复上下文
- 修复 `maybeCompact`：从仅日志改为实际触发 Compaction 流程
- 最大步数到达时注入 MAX_STEPS 警告消息并禁用工具，让模型优雅结束而非静默截断
- 添加 `_noop` 占位工具注入：当历史消息含 tool_calls 但当前工具列表为空时，注入 noop 工具避免部分 Provider 报错

### 增强

- 添加结构化输出支持（Structured Output）：当配置了 JSON Schema 时，循环使用结构化输出模式
- 添加重试状态回馈：重试时通过 Bus 发布 retry 事件，包含重试次数和预计等待时间

## Capabilities

### New Capabilities

（无新增独立能力模块）

### Modified Capabilities

- `react-loop`：流式重试、abort 传播、filterCompacted、maybeCompact 修复、MAX_STEPS 警告、noop 工具注入、结构化输出、重试状态反馈
- `builtin-tools`：工具名大小写修复增强（tool name repair）

## Impact

- `internal/runtime/engine.go`：主循环逻辑大幅增强（流式重试、abort、filterCompacted、maybeCompact、MAX_STEPS、noop 工具、结构化输出）
- `internal/llm/errors.go`：增加 retry-after 解析
- `internal/llm/provider.go`：可能需要 Usage 增加 CacheTokens 字段
- `internal/tool/router.go`：工具名大小写修复
- `internal/tools/compaction.go`：filterCompacted 逻辑
- `internal/store/store.go`：可能需要消息标记 compacted 状态
- 无破坏性变更，全部为增量补齐
