## ADDED Requirements

### Requirement: Leader Key 状态机

TUI MUST 实现 Leader key 机制。默认 Leader key 为 Ctrl+X。按下 Leader key 后进入 "leader 等待" 状态，持续 1.5 秒或直到按下后续键。

#### Scenario: Leader key 激活

- **WHEN** 用户按 Ctrl+X
- **THEN** TUI MUST 进入 leader 等待状态
- **AND** Footer MUST 显示 "-- LEADER --" 指示

#### Scenario: Leader key 超时

- **WHEN** leader 等待状态持续 1.5 秒无后续按键
- **THEN** TUI MUST 退出 leader 等待状态
- **AND** Footer MUST 恢复正常显示

#### Scenario: Leader + 后续键

- **WHEN** leader 等待状态中用户按 n
- **THEN** TUI MUST 执行 `<leader>n` 绑定的操作（新建会话）
- **AND** leader 等待状态 MUST 结束

### Requirement: Leader 快捷键映射

TUI MUST 支持以下 Leader 快捷键：
- `<leader>n`：新建会话
- `<leader>b`：切换侧边栏
- `<leader>a`：打开 Agent 选择 Dialog
- `<leader>l`：打开会话列表 Dialog
- `<leader>q`：退出

#### Scenario: leader+a Agent 选择

- **WHEN** 用户按 `<leader>a`
- **THEN** TUI MUST 打开 Select Dialog 显示可用 Agent 列表
- **AND** 选择后 MUST 通过 AgentSwitch 切换当前 session 的 Agent

### Requirement: 直接快捷键

以下快捷键 MUST 不需要 Leader 前缀，直接响应：
- `Ctrl+C`：退出
- `Enter`：发送消息（非 busy 且 input 有焦点时）
- `Escape`：关闭 dialog / 中断流式请求
- `PageUp` / `PageDown`：滚动 viewport

#### Scenario: Enter 发送

- **WHEN** 用户按 Enter 且 input 有焦点、非 busy、input 非空
- **THEN** TUI MUST 发送消息

#### Scenario: Escape 优先级

- **WHEN** 用户按 Escape
- **THEN** 优先级为：关闭 dialog > 中断流式 > 取消 leader 等待

### Requirement: 快捷键路由

键盘事件 MUST 按以下优先级路由：
1. Dialog 堆栈（最高优先级）
2. Leader key 状态机
3. 直接快捷键（Ctrl+C 等）
4. 活跃组件（input / sidebar）

#### Scenario: Dialog 中的键盘输入

- **WHEN** Dialog 堆栈非空且用户按 j
- **THEN** 输入 MUST 路由到 Dialog
- **AND** MUST NOT 传递到 input 组件

## MODIFIED Requirements

### Requirement: 模式切换

#### Scenario: Ctrl+P 保留

- **WHEN** 用户按 Ctrl+P
- **THEN** TUI MUST 打开命令面板或 Agent 选择 Dialog（替代现有的循环切换 mode）

### Requirement: 会话侧边栏

#### Scenario: 快捷键迁移

- **WHEN** 用户按 `<leader>b`（替代 Ctrl+B）
- **THEN** 侧边栏 MUST 切换显示/隐藏
