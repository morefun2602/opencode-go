# tui Specification

## Purpose

定义基于 Bubble Tea 框架的终端 UI，提供对话视图、会话管理、输入区域、工具确认与主题等交互体验。

## Requirements

### Requirement: Bubble Tea 应用框架

TUI MUST 基于 Bubble Tea 框架实现，提供完整的终端 UI 体验替代当前的原始 REPL。

#### Scenario: 启动 TUI

- **WHEN** 用户运行 `opencode-go tui` 或不带子命令启动
- **THEN** 系统 MUST 进入全屏终端 UI 模式

### Requirement: 对话视图

TUI MUST 提供对话视图，显示消息流（用户消息、助手消息、工具调用/结果），支持 Markdown 渲染和代码高亮。

#### Scenario: Markdown 渲染

- **WHEN** 助手消息包含 Markdown 格式文本
- **THEN** TUI MUST 使用 glamour 渲染 Markdown（标题、列表、代码块等）

#### Scenario: 工具调用可视化

- **WHEN** 助手消息包含 tool_call
- **THEN** TUI MUST 显示工具名、参数摘要和执行状态

### Requirement: 会话侧边栏

TUI MUST 提供可切换的会话侧边栏，列出当前工作区的会话（按时间倒序）。用户可以创建、选择和切换会话。

#### Scenario: 切换会话

- **WHEN** 用户在侧边栏选择另一个会话
- **THEN** 对话视图 MUST 加载该会话的消息历史

### Requirement: 输入区域

TUI MUST 提供多行文本输入区域，支持 Enter 发送、Shift+Enter 换行。输入区域 MUST 支持粘贴多行文本。

#### Scenario: 发送消息

- **WHEN** 用户在输入区域输入文本并按 Enter
- **THEN** 系统 MUST 将文本作为用户消息发送给 Engine

### Requirement: 工具确认对话框

当工具权限为 `ask` 时，TUI MUST 显示确认对话框，展示工具名和参数，等待用户确认或拒绝。

#### Scenario: 确认执行

- **WHEN** 确认对话框显示且用户按 y
- **THEN** 工具 MUST 执行

#### Scenario: 拒绝执行

- **WHEN** 确认对话框显示且用户按 n
- **THEN** 工具 MUST 返回拒绝结果，ReAct 循环继续

### Requirement: 模式切换

TUI MUST 提供快捷键（如 Ctrl+P）在 Agent 模式间切换，并在状态栏显示当前模式。

#### Scenario: 状态栏显示模式

- **WHEN** TUI 运行中
- **THEN** 底部状态栏 MUST 显示当前 Agent 模式名称

### Requirement: 主题

TUI MUST 支持至少 `dark`（默认）和 `light` 两种主题，通过配置或快捷键切换。

#### Scenario: 主题切换

- **WHEN** 用户通过配置设置 `theme: "light"`
- **THEN** TUI MUST 使用浅色配色方案
