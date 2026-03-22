## ADDED Requirements

### Requirement: Header 组件

TUI MUST 实现 Header 组件，位于对话区域顶部，显示：
- 会话标题（左侧，加粗）
- 当前 Agent 名称（中部或标题旁）
- 当前模型名称（右侧，subtle 色）

Header 高度 MUST 固定为 1 行。

#### Scenario: Header 信息显示

- **WHEN** TUI 运行中且有活跃会话
- **THEN** Header MUST 显示会话标题、Agent 名称和模型名称
- **AND** 若会话标题为空，MUST 显示 "New Session"

### Requirement: Footer 组件

TUI MUST 实现 Footer 组件，位于输入区域下方，替换当前的 statusBar。显示：
- 左侧：Agent 模式标签（带色块背景）
- 中部：错误信息（如有）或 busy 状态
- 右侧：可用快捷键提示

Footer 高度 MUST 固定为 1 行。

#### Scenario: Footer busy 状态

- **WHEN** Engine 正在处理请求
- **THEN** Footer 中部 MUST 显示 spinner 或 "thinking..." 指示

#### Scenario: Footer 错误

- **WHEN** 最近一次操作产生错误
- **THEN** Footer 中部 MUST 以 error 色显示错误摘要

#### Scenario: Footer leader key 状态

- **WHEN** 用户按了 Leader key 正在等待后续按键
- **THEN** Footer MUST 显示 "-- LEADER --" 指示

### Requirement: 布局结构

TUI 主布局 MUST 按以下结构组织：

```
┌───────────────────────────────┐
│ Header (1 line)               │
├───────────────────────────────┤
│                               │
│ Chat Viewport (flexible)      │
│                               │
├───────────────────────────────┤
│ Input (2-4 lines)             │
├───────────────────────────────┤
│ Footer (1 line)               │
└───────────────────────────────┘
```

当侧边栏打开时，布局 MUST 在左侧增加侧边栏列。

#### Scenario: 窗口尺寸适应

- **WHEN** 终端窗口调整大小
- **THEN** 各组件 MUST 重新计算尺寸
- **AND** Chat Viewport MUST 占据除 Header、Input、Footer 外的全部可用高度

## MODIFIED Requirements

### Requirement: 对话视图

#### Scenario: 布局集成

- **WHEN** TUI View() 被调用
- **THEN** 输出 MUST 按 Header → Viewport → Input → Footer 顺序纵向拼接
- **AND** 当前的 statusBar() 方法 MUST 被 Footer 组件替代
