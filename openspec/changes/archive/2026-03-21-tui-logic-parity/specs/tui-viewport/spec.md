## ADDED Requirements

### Requirement: Viewport 消息滚动

TUI MUST 使用 `bubbles/viewport` 组件包裹消息列表，替代当前的 lipgloss Height 截断渲染。viewport MUST 支持垂直滚动。

#### Scenario: 消息超出可视区域

- **WHEN** 消息列表总高度超过 viewport 可用高度
- **THEN** viewport MUST 允许用户滚动查看超出部分
- **AND** 默认 MUST 显示最底部内容（最新消息）

### Requirement: 键盘滚动

viewport MUST 支持以下滚动快捷键：
- PageUp / PageDown：翻页滚动
- Ctrl+U / Ctrl+D：半页滚动
- Home / End：跳转到顶部 / 底部

#### Scenario: PageDown 翻页

- **WHEN** 用户按 PageDown 且消息有更多内容在下方
- **THEN** viewport MUST 向下滚动一整页

### Requirement: Sticky Scroll（自动滚动到底部）

viewport MUST 实现 sticky scroll 行为：
- 当 viewport 已在底部时，新消息到达 MUST 自动滚动到底部
- 当用户手动向上滚动时，MUST 暂停自动滚动
- 用户滚动回底部时 MUST 恢复 sticky scroll

#### Scenario: 新消息自动滚动

- **WHEN** viewport 在底部且新的流式 chunk 到达
- **THEN** viewport MUST 自动滚动使最新内容可见

#### Scenario: 用户上滚暂停

- **WHEN** 用户按 PageUp 向上滚动
- **THEN** viewport MUST NOT 自动滚动到底部
- **AND** 新消息到达时 viewport 位置 MUST 保持不变

#### Scenario: 恢复 sticky scroll

- **WHEN** 用户滚动到底部（或按 End）
- **THEN** sticky scroll MUST 恢复
- **AND** 后续新消息 MUST 再次自动滚动到底部

### Requirement: 鼠标滚轮支持

viewport MUST 支持鼠标滚轮滚动（需启用 `tea.WithMouseAllMotion()` 或 `tea.WithMouseCellMotion()`）。

#### Scenario: 鼠标滚动

- **WHEN** 用户使用鼠标滚轮
- **THEN** viewport MUST 对应方向滚动消息列表

## MODIFIED Requirements

### Requirement: 对话视图

#### Scenario: viewport 集成

- **WHEN** chat 组件渲染消息列表
- **THEN** MUST 通过 viewport 组件包裹渲染输出
- **AND** MUST NOT 使用 lipgloss Height 截断
