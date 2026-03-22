# Design: TUI Logic Parity

## D1: 流式渲染架构

**决策**：使用 `tea.Program` 引用注入 + goroutine 回调模式。

`streamMessage` 命令启动 goroutine 调用 `CompleteTurnStream`，通过闭包捕获 `*tea.Program` 指针，在 stream 回调中调用 `p.Send(streamChunk{text})` 实时发送增量消息。

```go
type streamChunk struct{ text string }
type streamDone  struct{ err error  }

func streamMessage(p *tea.Program, eng *runtime.Engine, ws, session, text string) tea.Cmd {
    return func() tea.Msg {
        ctx, cancel := context.WithCancel(context.Background())
        // 将 cancel 存入 Model 以便 Escape 中断
        p.Send(streamStarted{cancel: cancel})
        err := eng.CompleteTurnStream(ctx, ws, session, text, func(chunk string) error {
            p.Send(streamChunk{text: chunk})
            return nil
        })
        return streamDone{err: err}
    }
}
```

Model 维护 `streamBuf strings.Builder` 累积 chunk，每个 `streamChunk` 触发 View 重绘。`streamDone` 时清空 buf 并加载最终消息。

**理由**：Bubble Tea 的 Cmd 返回单个 Msg，无法做多次回调。`p.Send()` 是官方推荐的外部事件注入方式。

## D2: Part 解析与渲染管道

**决策**：在 `parts.go` 中实现消息→渲染块的转换管道。

```go
type RenderBlock struct {
    Type    string // "text", "tool_call", "tool_result", "user", "system"
    Content string // 渲染后的 lipgloss 字符串
}

func BuildRenderBlocks(msgs []store.MessageRow, theme Theme, width int) []RenderBlock
```

转换流程：
1. 遍历 messages，按 role 分类
2. assistant 消息：解析 Parts JSON → 对每个 Part 调用类型专用渲染函数
3. user 消息：渲染为 "You" + body
4. tool 消息：跳过（其 result 已通过 tool_call 卡片的配对显示）

tool_call 与 tool_result 配对：遍历消息列表，为每个 tool_call Part 查找后续的 role=tool 消息中 ToolCallID 匹配的 result。

**理由**：预渲染为 RenderBlock 列表，viewport 只需 Join + SetContent，无需每帧重算。

## D3: 工具卡片渲染

**决策**：在 `tool_card.go` 中实现工具类型→渲染函数的分发表。

```go
var toolRenderers = map[string]func(name string, args map[string]any, result string, isErr bool, w int, theme Theme) string{
    "bash":  renderBash,
    "read":  renderRead,
    "edit":  renderEdit,
    "write": renderWrite,
    "grep":  renderGrep,
    "glob":  renderGlob,
}

func RenderToolCard(name string, args map[string]any, result string, isErr bool, w int, theme Theme) string
```

通用 fallback：显示工具名 + args JSON 前 60 字符。
结果折叠：超过 5 行时截断为 3 行 + "▸ N more lines"。

卡片样式：使用 lipgloss.Border + theme.Border 色。头部区域背景色区分工具类型。

**理由**：分发表模式易于扩展新工具渲染器。

## D4: Dialog 堆栈系统

**决策**：在 `dialog.go` 中实现 Dialog 接口 + 堆栈管理。

```go
type Dialog interface {
    Update(msg tea.Msg) (Dialog, tea.Cmd)
    View(w, h int, theme Theme) string
    Done() bool
    Result() any
}

type DialogStack struct {
    stack []Dialog
}

func (ds *DialogStack) Push(d Dialog)
func (ds *DialogStack) Pop() Dialog
func (ds *DialogStack) Top() Dialog
func (ds *DialogStack) Empty() bool
```

Dialog 渲染：在 Model.View() 末尾，若 `!dialogStack.Empty()`，则在主内容之上叠加渲染。居中框使用 lipgloss.Place()。

Confirm Dialog 的 `eng.Confirm` 集成：使用 channel 同步。

```go
type confirmRequest struct {
    name string
    args map[string]any
    ch   chan bool
}
```

`eng.Confirm` 发送 `confirmRequest` 给 TUI，TUI 弹出 Confirm Dialog，用户响应后写入 channel。由于 `eng.Confirm` 在 Engine goroutine 中调用，需通过 `p.Send()` 传递到 TUI 的 Update 循环。

**理由**：channel 同步是 Go 中标准的 goroutine 间通信模式，结合 `p.Send()` 实现 Engine↔TUI 跨 goroutine 对话框交互。

## D5: Header 与 Footer 组件

**决策**：在 `header.go` 和 `footer.go` 中实现为纯函数（无 Model 状态）。

```go
func RenderHeader(title, agent, model string, w int, theme Theme) string
func RenderFooter(mode string, err error, busy bool, leaderActive bool, hints string, w int, theme Theme) string
```

Header：
- 左侧：会话标题（加粗，宽度 50%）
- 右侧：Agent 名称 + 模型名称（subtle 色，右对齐）
- 背景色：theme.Border

Footer：
- 左侧：mode 标签（带背景色块）
- 中部：err / busy 状态 / leader 指示
- 右侧：快捷键提示
- 背景色：theme.Border

**理由**：Header/Footer 无交互状态，纯函数渲染最简洁。状态由 Model 传入。

## D6: Viewport 集成

**决策**：使用 `github.com/charmbracelet/bubbles/viewport` 作为消息区域容器。

```go
type chatModel struct {
    viewport viewport.Model
    sticky   bool // auto-scroll to bottom
}
```

Sticky scroll 逻辑：
- 初始化时 `sticky = true`
- 每次 SetContent 或 streamChunk 时，若 sticky，调用 `viewport.GotoBottom()`
- 用户按 PageUp/Up 时设 `sticky = false`
- 用户按 End 或 viewport 已在底部时设 `sticky = true`

渲染流程：
1. `BuildRenderBlocks()` 生成渲染块列表
2. `strings.Join()` 拼接为完整内容
3. 流式 chunk 时追加到末尾
4. `viewport.SetContent(content)` 更新
5. 若 sticky，`viewport.GotoBottom()`

鼠标支持：在 `tea.NewProgram` 中添加 `tea.WithMouseCellMotion()`。

**理由**：bubbles/viewport 是 Bubble Tea 官方推荐的滚动容器，已处理好键盘/鼠标事件路由。

## D7: Leader Key 状态机

**决策**：在 `leader.go` 中实现简单的两状态状态机。

```go
type LeaderState struct {
    active  bool
    timer   *time.Timer
    timeout time.Duration // 默认 1.5s
}

func (l *LeaderState) Activate() tea.Cmd   // 设 active=true，启动 timeout timer
func (l *LeaderState) Deactivate()          // 设 active=false
func (l *LeaderState) IsActive() bool

type leaderTimeout struct{} // timeout Msg
```

快捷键映射表：

```go
var leaderBindings = map[string]func(m *Model) tea.Cmd{
    "n": func(m *Model) tea.Cmd { return createSession(m.store, m.workspace) },
    "b": func(m *Model) tea.Cmd { m.toggleSidebar(); return nil },
    "a": func(m *Model) tea.Cmd { return m.openAgentDialog() },
    "l": func(m *Model) tea.Cmd { return m.openSessionDialog() },
    "q": func(m *Model) tea.Cmd { return tea.Quit },
}
```

在 `handleKey` 中：若 `leader.IsActive()` 且 msg.String() 在 leaderBindings 中，执行对应操作并 Deactivate。

**理由**：TS 版的 Leader key 体验对终端 power user 非常自然。简单状态机 + map 实现开销极低。

## D8: Model 重构

**决策**：顶层 Model 结构扩展为：

```go
type Model struct {
    engine    *runtime.Engine
    store     store.Store
    workspace string
    theme     Theme
    width, height int
    active    focus
    program   *tea.Program // 流式回调需要

    // 组件
    chat     chatModel     // 内含 viewport
    sidebar  sidebarModel
    input    inputModel
    dialogs  DialogStack
    leader   LeaderState

    // 状态
    sessions  []store.SessionRow
    session   string
    messages  []store.MessageRow
    busy      bool
    streaming bool
    streamBuf strings.Builder
    streamCancel context.CancelFunc
    err       error

    // Agent 信息（来自 Engine）
    agentName string
    modelName string
}
```

`program` 字段通过 `Init()` 中延迟设置或通过构造函数注入。

Update 优先级：
1. Dialog 堆栈（若非空）
2. confirmRequest 处理
3. Leader key 状态机
4. 直接快捷键
5. 活跃组件

**理由**：集中管理所有状态，清晰的优先级链避免事件冲突。
