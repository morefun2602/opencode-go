## 1. Theme 扩展（D5, D3）

- [x] 1.1 修改 `internal/tui/theme.go`：Theme 结构体新增 `ToolBorder`、`ToolHeader`、`DialogOverlay`、`HeaderBg`、`FooterBg` 字段
- [x] 1.2 更新 Dark 和 Light 主题赋值

## 2. Viewport 集成（D6）

- [x] 2.1 添加 `github.com/charmbracelet/bubbles` viewport 依赖（若尚未有）
- [x] 2.2 修改 `internal/tui/chat.go`：chatModel 新增 `viewport.Model` 字段和 `sticky bool`
- [x] 2.3 实现 `chatModel.SetContent(content string)` 方法：设置 viewport 内容，sticky 时 GotoBottom
- [x] 2.4 实现 sticky scroll 逻辑：PageUp/Up 关闭 sticky，End/GotoBottom 恢复 sticky
- [x] 2.5 修改 `chatModel.View()`：返回 `viewport.View()` 替代 lipgloss Height 截断

## 3. Part 解析与渲染（D2）

- [x] 3.1 新建 `internal/tui/parts.go`：定义 `RenderBlock` 结构体和 `BuildRenderBlocks` 函数
- [x] 3.2 实现 Parts JSON 解析：`store.MessageRow.Parts` → `[]llm.Part`
- [x] 3.3 实现 user 消息渲染：角色前缀 + body
- [x] 3.4 实现 assistant text Part 渲染：glamour Markdown
- [x] 3.5 实现 tool_call / tool_result 配对逻辑：通过 ToolCallID 关联
- [x] 3.6 移除 500 字符截断，改用完整渲染 + viewport 滚动
- [x] 3.7 修改 `chatModel.View()`：调用 `BuildRenderBlocks` 生成内容

## 4. 工具调用卡片（D3）

- [x] 4.1 新建 `internal/tui/tool_card.go`：定义 `RenderToolCard` 函数和 `toolRenderers` 分发表
- [x] 4.2 实现通用卡片 fallback：工具名 + args 摘要 + 状态图标
- [x] 4.3 实现 `renderBash`：命令行 + 输出前 3 行
- [x] 4.4 实现 `renderRead`：文件路径 + "(N lines read)"
- [x] 4.5 实现 `renderEdit` / `renderWrite`：文件路径 + 状态
- [x] 4.6 实现 `renderGrep` / `renderGlob`：pattern + 匹配数
- [x] 4.7 实现结果折叠：>5 行截断为 3 行 + "▸ N more lines"

## 5. Header 与 Footer（D5）

- [x] 5.1 新建 `internal/tui/header.go`：实现 `RenderHeader(title, agent, model, w, theme) string`
- [x] 5.2 新建 `internal/tui/footer.go`：实现 `RenderFooter(mode, err, busy, leaderActive, hints, w, theme) string`
- [x] 5.3 修改 `model.go` View()：使用 Header + Viewport + Input + Footer 布局替代现有 chatContent + inputContent + statusBar
- [x] 5.4 移除 `statusBar()` 方法

## 6. Dialog 系统（D4）

- [x] 6.1 新建 `internal/tui/dialog.go`：定义 Dialog 接口、DialogStack 结构体
- [x] 6.2 实现 Confirm Dialog：标题 + 描述 + y/n 响应
- [x] 6.3 实现 Select Dialog：标题 + 列表 + j/k 导航 + Enter 选择
- [x] 6.4 实现 Alert Dialog：标题 + 内容 + 任意键关闭
- [x] 6.5 修改 `model.go` View()：Dialog 非空时在主内容上叠加 Dialog 渲染
- [x] 6.6 修改 `model.go` Update()：Dialog 非空时键盘事件路由到 Dialog

## 7. Engine Confirm 集成（D4）

- [x] 7.1 定义 `confirmRequest` 结构体（name、args、ch chan bool）
- [x] 7.2 修改 `internal/cli/tui.go`：eng.Confirm 改为通过 `p.Send(confirmRequest{...})` 请求确认
- [x] 7.3 修改 `model.go` Update()：收到 `confirmRequest` 时 push Confirm Dialog，Dialog 完成时向 channel 写入结果
- [ ] 7.4 测试：确保 Confirm Dialog y→true、n→false、ESC→false

## 8. 流式渲染（D1）

- [x] 8.1 修改 `internal/tui/commands.go`：新增 `streamMessage` 函数，使用 `p.Send()` 发送 streamChunk
- [x] 8.2 定义消息类型：`streamStarted{cancel}`、`streamChunk{text}`、`streamDone{err}`
- [x] 8.3 修改 Model：新增 `streaming bool`、`streamBuf strings.Builder`、`streamCancel context.CancelFunc`、`program *tea.Program`
- [x] 8.4 修改 `model.go` Update()：处理 streamChunk（追加 buf + 刷新 viewport）、streamDone（清空 buf + reload）
- [x] 8.5 修改 `handleKey`：Enter 时调用 `streamMessage` 替代 `sendMessage`
- [x] 8.6 修改 `chatModel`：streaming 时在 viewport 底部追加 streamBuf 内容
- [x] 8.7 实现 Escape 中断：streaming 时按 Escape 调用 `streamCancel()`

## 9. Leader Key（D7）

- [x] 9.1 新建 `internal/tui/leader.go`：定义 `LeaderState` 结构体和 `leaderBindings` map
- [x] 9.2 实现 `Activate()` tea.Cmd（启动 timeout timer）、`Deactivate()`、`IsActive()`
- [x] 9.3 定义 `leaderTimeout` 消息类型
- [x] 9.4 修改 `handleKey`：Ctrl+X 激活 leader；leader active 时查 leaderBindings
- [x] 9.5 实现 leader bindings：n（新会话）、b（侧边栏）、a（Agent Dialog）、l（会话 Dialog）、q（退出）
- [x] 9.6 修改 `model.go` Update()：处理 `leaderTimeout` 消息

## 10. Model 重构与布局整合（D8）

- [x] 10.1 修改 `Model` 结构体：新增 dialogs、leader、streaming 相关字段
- [x] 10.2 修改 `New()` 构造函数：初始化新增组件
- [x] 10.3 修改 `Init()`：若需要传入 `*tea.Program`，通过 Init 消息延迟设置
- [x] 10.4 修改 Update() 优先级链：Dialog → confirmRequest → Leader → 直接快捷键 → 组件
- [x] 10.5 修改 View() 布局：Header + viewport + Input + Footer + Dialog overlay
- [x] 10.6 修改 `internal/cli/tui.go`：添加 `tea.WithMouseCellMotion()` 启用鼠标滚动

## 11. 集成验证

- [x] 11.1 运行 `go build ./...` 确保编译通过
- [x] 11.2 运行 `go vet ./internal/tui/...` 检查代码质量
- [ ] 11.3 手动验证：启动 TUI，发送消息，验证流式渲染
- [ ] 11.4 手动验证：工具调用卡片显示
- [ ] 11.5 手动验证：PageUp/PageDown 滚动
- [ ] 11.6 手动验证：Ctrl+X 后按 n 新建会话
- [ ] 11.7 手动验证：Escape 中断流式请求
