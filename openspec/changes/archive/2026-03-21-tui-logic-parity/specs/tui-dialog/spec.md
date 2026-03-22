## ADDED Requirements

### Requirement: Dialog 堆栈框架

TUI MUST 实现模态 Dialog 堆栈系统。Dialog MUST：
- 在主内容之上渲染半透明覆盖层
- 支持 Push（添加新 dialog）和 Pop（关闭顶部 dialog）
- ESC 键 MUST 关闭顶部 dialog
- 当 dialog 堆栈非空时，键盘输入 MUST 仅路由到顶部 dialog

#### Scenario: Dialog 覆盖渲染

- **WHEN** Dialog 堆栈包含一个 Confirm dialog
- **THEN** TUI MUST 在主内容之上渲染居中的 dialog 框
- **AND** 主内容区域 MUST 保持可见但不可交互

#### Scenario: ESC 关闭 Dialog

- **WHEN** 用户按 ESC 且 dialog 堆栈非空
- **THEN** 顶部 dialog MUST 被关闭
- **AND** 焦点 MUST 恢复到之前的组件

#### Scenario: 多层 Dialog

- **WHEN** 堆栈中有两个 dialog
- **THEN** 仅顶部 dialog 可接收输入
- **AND** 按 ESC 关闭顶部后，第二个 dialog MUST 变为活跃

### Requirement: Confirm Dialog

TUI MUST 实现 Confirm Dialog，显示标题、描述文本和 y/n 选项。

#### Scenario: 确认操作

- **WHEN** Confirm Dialog 显示且用户按 y
- **THEN** Dialog MUST 关闭并返回 true

#### Scenario: 拒绝操作

- **WHEN** Confirm Dialog 显示且用户按 n 或 ESC
- **THEN** Dialog MUST 关闭并返回 false

### Requirement: Select Dialog

TUI MUST 实现 Select Dialog，显示标题和可选项列表。支持 j/k 或上下键导航，Enter 选择。

#### Scenario: 列表选择

- **WHEN** Select Dialog 显示包含 5 个选项的列表
- **THEN** 用户 MUST 能用 j/k 导航
- **AND** 按 Enter 选择当前高亮项
- **AND** 选择后 Dialog MUST 关闭并返回选中值

### Requirement: Alert Dialog

TUI MUST 实现 Alert Dialog，显示标题和信息文本。按任意键或 ESC 关闭。

#### Scenario: 信息展示

- **WHEN** Alert Dialog 显示
- **THEN** 用户按任意键 MUST 关闭 Dialog

### Requirement: Engine Confirm 集成

TUI MUST 将 `Engine.Confirm` 注入为通过 Dialog 系统进行交互确认的函数。当 Agent permission 为 `ask` 或 Policy 要求 `ask` 时，MUST 弹出 Confirm Dialog 等待用户响应。

#### Scenario: 工具权限确认

- **WHEN** Engine 执行工具时 permission 为 ask
- **THEN** TUI MUST 弹出 Confirm Dialog 显示工具名和参数
- **AND** 用户选择 y 后工具 MUST 执行
- **AND** 用户选择 n 后工具 MUST 返回拒绝结果

## MODIFIED Requirements

### Requirement: 工具确认对话框

#### Scenario: TUI Confirm 替代默认

- **WHEN** TUI 启动时
- **THEN** `eng.Confirm` MUST 被设置为 Dialog 驱动的确认函数（替代当前的 `return true, nil`）
