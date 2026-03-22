## ADDED Requirements

### Requirement: 流式消息发送

TUI MUST 使用 `Engine.CompleteTurnStream` 替代 `Engine.CompleteTurn` 发送用户消息。流式回调 MUST 通过 Bubble Tea 的 `tea.Program.Send()` 将每个 chunk 发送为 `streamChunk` 消息，实现实时界面更新。

#### Scenario: 流式渲染 assistant 回复

- **WHEN** 用户发送消息后 Engine 开始流式返回
- **THEN** TUI MUST 在每个 chunk 到达时立即更新 assistant 消息区域的显示
- **AND** 用户 MUST 能看到文本逐步出现，而非等待整轮完成

#### Scenario: 流式状态指示

- **WHEN** 流式渲染进行中
- **THEN** TUI MUST 显示视觉指示（如闪烁光标或 spinner）
- **AND** 用户按 Escape MUST 能中断当前流式请求

### Requirement: streamMessage 命令

`commands.go` MUST 新增 `streamMessage` 命令，该命令 MUST：
1. 在 goroutine 中调用 `Engine.CompleteTurnStream`
2. 通过 `tea.Program.Send()` 将每个 text chunk 以 `streamChunk{text string}` 消息发送给 Model
3. 流式结束时发送 `streamDone{err error}` 消息
4. 支持通过 context cancel 中断

#### Scenario: 流式 chunk 传递

- **WHEN** Engine 流式回调被调用并传入 text chunk
- **THEN** `streamChunk` 消息 MUST 立即被发送到 Bubble Tea Program
- **AND** Model 的 Update 方法 MUST 将 chunk 追加到当前 streaming buffer

#### Scenario: 流式完成

- **WHEN** Engine 流式调用完成（成功或失败）
- **THEN** `streamDone` 消息 MUST 被发送
- **AND** Model MUST 将 streaming buffer 提交为最终消息并重新加载消息列表

### Requirement: 中断支持

TUI MUST 允许用户在流式渲染期间按 Escape 中断当前请求。中断 MUST 通过取消传给 `CompleteTurnStream` 的 context 实现。

#### Scenario: Escape 中断流式

- **WHEN** 用户在流式渲染期间按 Escape
- **THEN** Engine MUST 停止当前 LLM 调用
- **AND** 已收到的部分回复 MUST 保留显示
- **AND** TUI MUST 恢复到可输入状态

## MODIFIED Requirements

### Requirement: 对话视图

#### Scenario: 实时更新

- **WHEN** 流式 chunk 到达
- **THEN** 对话视图 MUST 在当前 assistant 消息末尾追加新文本
- **AND** viewport MUST 自动滚动到底部（sticky scroll 激活时）
