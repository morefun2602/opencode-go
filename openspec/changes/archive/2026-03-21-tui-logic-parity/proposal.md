# Proposal: TUI Logic Parity

## Why

Go 版 TUI 当前仅实现了最基础的对话界面：4 个组件（chat、input、sidebar、status bar），使用 `eng.CompleteTurn` 同步阻塞等待结果，工具确认始终返回 true，消息渲染仅区分 role 角色且截断 500 字符，无任何对话框、通知、工具可视化或流式更新能力。

TypeScript 参考实现拥有 30+ 组件、流式 part 渲染、堆栈式对话框系统、结构化工具调用可视化、Toast 通知、`@` 文件补全、Leader key 体系、30+ 主题等完整交互能力。

Go 版需要逐步补齐核心 TUI 交互能力，以提供接近 TS 版的用户体验。本次变更聚焦于最高优先级的 7 个能力域。

## What Changes

### 1. 流式渲染 — 从同步阻塞到实时更新
- 将 `sendMessage` 从调用 `CompleteTurn` 改为 `CompleteTurnStream`
- 引入 `tea.Msg` 增量更新机制：每收到 chunk 时发送 `streamChunk` 消息，TUI 实时刷新 assistant 回复
- 新增 streaming 状态管理（当前 part 缓冲、光标闪烁等）

### 2. 消息 Part 模型 — 结构化消息渲染
- 解析 `store.MessageRow.Parts` JSON 为 `llm.Part` 切片
- 按 Part 类型分发渲染：text（Markdown）、tool_call（工具调用卡片）、tool_result（执行结果）
- assistant 消息不再截断为 500 字符，改用 viewport 滚动

### 3. 工具调用可视化 — 结构化工具卡片
- 为核心工具实现专用渲染组件：bash（命令+输出）、read（文件路径）、edit/write（文件路径+diff 摘要）、grep/glob（搜索模式+匹配数）
- 工具调用显示：工具名 + 关键参数摘要 + 执行状态（running/done/error）
- 折叠过长输出，默认显示摘要

### 4. 对话框系统 — 堆栈式 Dialog 框架
- 实现 `Dialog` 基础框架：模态覆盖、ESC 关闭、堆栈管理（push/pop）
- 实现 3 种基础对话框：Confirm（y/n）、Select（列表选择）、Alert（信息展示）
- 集成 `eng.Confirm`：工具权限为 ask 时弹出 Confirm Dialog

### 5. Header 与 Footer — 结构化布局
- Header：显示会话标题、当前模型名称、Agent 名称、token 消耗
- Footer：显示快捷键提示、模式指示、busy 状态、错误信息
- 替换当前的单行 statusBar

### 6. Viewport 滚动 — 消息区域滚动支持
- 使用 `bubbles/viewport` 替代当前的 lipgloss Height 截断渲染
- 支持 PageUp/PageDown、鼠标滚轮滚动
- 新消息到达时自动滚动到底部（sticky scroll）
- 用户手动滚动时暂停自动滚动

### 7. Leader Key 快捷键体系
- 引入 Leader key 概念（默认 Ctrl+X），按 Leader 后进入快捷键等待状态
- 迁移现有快捷键到 Leader 体系：`<leader>n` 新建会话、`<leader>b` 侧边栏、`<leader>a` Agent 列表
- 保留 Ctrl+C 退出、Enter 发送等直接快捷键
- 在 Footer 显示 Leader key 状态

## Capabilities

- tui-streaming
- tui-parts-rendering
- tui-tool-cards
- tui-dialog
- tui-layout
- tui-viewport
- tui-keybinds

## Impact

### 修改文件
- `internal/tui/model.go` — 顶层 Model 重构：添加 dialog 栈、viewport、streaming 状态、leader key 状态
- `internal/tui/chat.go` — 重写为 Part-based 渲染 + viewport 集成
- `internal/tui/input.go` — 适配新布局
- `internal/tui/sidebar.go` — 适配新布局
- `internal/tui/theme.go` — 扩展主题字段（ToolBorder、DialogOverlay 等）
- `internal/tui/commands.go` — 新增 streamMessage 命令 + 增量消息
- `internal/cli/tui.go` — Confirm 注入改为 Dialog 交互

### 新增文件
- `internal/tui/dialog.go` — Dialog 框架 + Confirm/Select/Alert
- `internal/tui/header.go` — Header 组件
- `internal/tui/footer.go` — Footer 组件
- `internal/tui/viewport.go` — Viewport 封装（消息滚动）
- `internal/tui/tool_card.go` — 工具调用卡片渲染
- `internal/tui/parts.go` — Part 解析与分发渲染
- `internal/tui/leader.go` — Leader key 状态机
