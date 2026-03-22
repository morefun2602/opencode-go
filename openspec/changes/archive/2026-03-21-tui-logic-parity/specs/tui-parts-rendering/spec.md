## ADDED Requirements

### Requirement: Part 解析

TUI MUST 解析 `store.MessageRow.Parts` 字段（JSON 字符串）为 `[]llm.Part` 切片。对于 Parts 为空但 Body 非空的消息，MUST 回退为单个 text part。

#### Scenario: Parts JSON 解析

- **WHEN** 加载消息历史且 MessageRow.Parts 为非空 JSON
- **THEN** TUI MUST 将其反序列化为 Part 切片
- **AND** 按 Part 类型分发到对应渲染组件

#### Scenario: 无 Parts 回退

- **WHEN** MessageRow.Parts 为空字符串或 "[]"
- **THEN** TUI MUST 使用 MessageRow.Body 作为单个 text part 渲染

### Requirement: Part 类型分发渲染

assistant 消息 MUST 按 Part 类型分发渲染，而非将整个 Body 作为单块 Markdown 渲染：
- `text` Part → Markdown 渲染（glamour）
- `tool_call` Part → 工具调用卡片（参见 tui-tool-cards spec）
- `tool_result` Part → 仅在与 tool_call 配对时显示结果摘要

#### Scenario: 混合 Part 渲染

- **WHEN** assistant 消息包含 [text, tool_call, text] 三个 part
- **THEN** TUI MUST 按序渲染：Markdown 段落 → 工具调用卡片 → Markdown 段落

### Requirement: user 消息渲染

user 消息 MUST 显示为带角色前缀的纯文本块，前缀样式使用 theme.Primary 色。

#### Scenario: 用户消息显示

- **WHEN** 渲染 user 角色消息
- **THEN** MUST 显示 "You" 前缀 + 消息文本
- **AND** 前缀 MUST 使用 primary 颜色加粗

### Requirement: tool 消息渲染

role=tool 消息 MUST 不单独显示为独立消息块。其内容 MUST 通过 tool_call 卡片的结果区域展示（与对应的 tool_call Part 配对）。

#### Scenario: tool 消息折叠

- **WHEN** 渲染消息列表
- **THEN** role=tool 的消息 MUST NOT 作为独立行显示
- **AND** 其 result 内容 MUST 关联到前一个 assistant 消息的对应 tool_call 卡片中

### Requirement: 移除 500 字符截断

assistant 消息 MUST NOT 再截断为 500 字符。完整内容 MUST 通过 viewport 滚动访问。

#### Scenario: 长消息完整显示

- **WHEN** assistant 消息超过 500 字符
- **THEN** 消息 MUST 完整渲染
- **AND** 用户 MUST 能通过滚动查看全部内容
