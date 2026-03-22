package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/morefun2602/opencode-go/internal/runtime"
	"github.com/morefun2602/opencode-go/internal/store"
)

type focus int

const (
	focusChat focus = iota
	focusSidebar
)

type route int

const (
	routeHome route = iota
	routeSession
)

// Model is the top-level Bubble Tea model for the TUI application.
type Model struct {
	engine    *runtime.Engine
	store     store.Store
	workspace string
	theme     Theme
	width     int
	height    int
	active    focus
	route     route
	mode      string
	program   *tea.Program

	// Components
	chat         chatModel
	sidebar      sidebarModel
	input        inputModel
	dialogs      DialogStack
	leader       LeaderState
	spinner      spinner.Model
	autocomplete autocompleteModel
	palette      commandPalette
	toasts       ToastManager
	mention      mentionModel

	// Session state
	sessions []store.SessionRow
	session  string
	messages []store.MessageRow
	busy     bool
	err      error

	// Streaming state
	streaming    bool
	streamBuf    strings.Builder
	streamCancel context.CancelFunc

	// Agent/model display info
	agentName string
	modelName string
}

// New creates a new top-level TUI model.
func New(eng *runtime.Engine, st store.Store, workspace string, theme Theme) Model {
	modelName := ""
	if eng.Agent.Model != "" {
		modelName = eng.Agent.Model
	} else if eng.Router != nil {
		ref := eng.Router.DefaultModel()
		if ref.ModelID != "" {
			if ref.ProviderID != "" {
				modelName = ref.ProviderID + "/" + ref.ModelID
			} else {
				modelName = ref.ModelID
			}
		}
	}

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(theme.Primary)

	slashCmds := defaultSlashCommands()

	return Model{
		engine:       eng,
		store:        st,
		workspace:    workspace,
		theme:        theme,
		mode:         "build",
		route:        routeHome,
		active:       focusChat,
		chat:         newChat(theme, 80, 20),
		sidebar:      newSidebar(theme),
		input:        newInput(theme),
		leader:       NewLeaderState(),
		spinner:      sp,
		autocomplete: newAutocomplete(slashCmds),
		palette:      newCommandPalette(),
		toasts:       newToastManager(),
		mention:      newMentionModel(workspace),
		agentName:    eng.Agent.Name,
		modelName:    modelName,
	}
}

// SetProgram injects the *tea.Program reference needed for streaming.
func (m *Model) SetProgram(p *tea.Program) {
	m.program = p
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, loadSessions(m.store, m.workspace))
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalcLayout()
		return m, nil

	case tea.KeyMsg:
		// Command palette takes top priority
		if m.palette.visible {
			action, consumed := m.palette.Update(msg)
			if action != nil {
				cmd := action()
				return m, cmd
			}
			if consumed {
				return m, nil
			}
		}
		if !m.dialogs.Empty() {
			return m.updateDialog(msg)
		}
		return m.handleKey(msg)

	case confirmRequest:
		return m.handleConfirmRequest(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case toastExpired:
		m.toasts.Expire()
		return m, nil

	case streamStarted:
		m.streamCancel = msg.cancel
		m.streaming = true
		m.streamBuf.Reset()
		return m, nil

	case streamChunk:
		m.streamBuf.WriteString(msg.text)
		m.chat.Refresh(m.messages, m.streamBuf.String(), m.theme)
		return m, nil

	case streamDone:
		m.streaming = false
		m.busy = false
		m.streamCancel = nil
		if msg.err != nil {
			m.err = msg.err
			cmd := m.toasts.Add("Error: "+msg.err.Error(), ToastError, 5*time.Second)
			return m, tea.Batch(loadMessages(m.store, m.workspace, m.session), cmd)
		}
		m.streamBuf.Reset()
		return m, loadMessages(m.store, m.workspace, m.session)

	case leaderTimeout:
		m.leader.Deactivate()
		return m, nil

	case sessionsLoaded:
		m.sessions = msg.sessions
		if len(m.sessions) > 0 && m.session == "" {
			m.session = m.sessions[0].ID
			return m, loadMessages(m.store, m.workspace, m.session)
		}
		return m, nil

	case sessionCreated:
		m.session = msg.id
		m.messages = nil
		m.route = routeSession
		m.chat.Refresh(nil, "", m.theme)
		cmd := m.toasts.Add("New session created", ToastSuccess, 2*time.Second)
		return m, tea.Batch(loadSessions(m.store, m.workspace), cmd)

	case messagesLoaded:
		m.messages = msg.messages
		m.chat.Refresh(m.messages, "", m.theme)
		if len(m.messages) > 0 && m.route == routeHome {
			m.route = routeSession
		}
		return m, nil

	case turnComplete:
		m.busy = false
		m.err = msg.err
		if msg.err != nil {
			cmd := m.toasts.Add("Error: "+msg.err.Error(), ToastError, 5*time.Second)
			return m, tea.Batch(loadMessages(m.store, m.workspace, m.session), cmd)
		}
		return m, loadMessages(m.store, m.workspace, m.session)

	case sessionSelected:
		m.session = msg.id
		m.route = routeSession
		return m, loadMessages(m.store, m.workspace, m.session)
	}

	// Forward other messages (mouse, etc.) to appropriate components
	if !m.dialogs.Empty() {
		return m.updateDialog(msg)
	}

	if m.route == routeSession {
		cmd := m.chat.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Mention popup takes priority when visible
	if m.mention.visible {
		selected, consumed := m.mention.Update(msg)
		if selected != nil {
			text := m.input.Value()
			atIdx := strings.LastIndex(text, "@")
			if atIdx >= 0 {
				m.input.area.SetValue(text[:atIdx] + "@" + selected.Value + " ")
				m.input.area.CursorEnd()
			}
			return m, nil
		}
		if consumed {
			return m, nil
		}
	}

	// Autocomplete takes priority when visible
	if m.autocomplete.visible {
		selected, consumed := m.autocomplete.Update(msg)
		if selected != nil {
			m.input.Reset()
			return m, m.executeSlashCommand(selected)
		}
		if consumed {
			return m, nil
		}
	}

	if m.leader.IsActive() {
		m.leader.Deactivate()
		if cmd := m.dispatchLeader(key); cmd != nil {
			return m, cmd
		}
		return m, nil
	}

	switch key {
	case "ctrl+c":
		return m, tea.Quit

	case "ctrl+x":
		return m, m.leader.Activate()

	case "ctrl+p":
		m.openCommandPalette()
		return m, nil

	case "esc":
		if m.autocomplete.visible {
			m.autocomplete.Hide()
			return m, nil
		}
		if m.mention.visible {
			m.mention.Hide()
			return m, nil
		}
		if m.streaming && m.streamCancel != nil {
			m.streamCancel()
			return m, nil
		}
		if m.active == focusSidebar {
			m.active = focusChat
			return m, nil
		}
		if m.route == routeSession && len(m.messages) == 0 {
			m.route = routeHome
			return m, nil
		}
		return m, nil

	case "pgup":
		m.chat.PageUp()
		return m, nil

	case "pgdown":
		m.chat.PageDown()
		return m, nil

	case "end":
		m.chat.GotoBottom()
		return m, nil

	case "enter":
		if m.route == routeHome {
			text := m.input.Value()
			if text == "" {
				m.route = routeSession
				if m.session == "" {
					return m, createSession(m.store, m.workspace)
				}
				return m, nil
			}
			m.input.Reset()
			m.autocomplete.Hide()
			m.mention.Hide()
			m.busy = true
			m.err = nil
			m.route = routeSession
			if m.session == "" {
				return m, createSessionAndSend(m.store, m.engine, m.workspace, text, m.program)
			}
			if m.program != nil {
				return m, streamMessage(m.program, m.engine, m.workspace, m.session, text)
			}
			return m, sendMessage(m.engine, m.workspace, m.session, text)
		}

		if m.active == focusChat && !m.busy && !m.streaming {
			text := m.input.Value()
			if text == "" {
				return m, nil
			}
			m.input.Reset()
			m.autocomplete.Hide()
			m.mention.Hide()
			m.busy = true
			m.err = nil
			if m.program != nil {
				return m, streamMessage(m.program, m.engine, m.workspace, m.session, text)
			}
			return m, sendMessage(m.engine, m.workspace, m.session, text)
		}
	}

	if m.active == focusSidebar {
		var cmd tea.Cmd
		m.sidebar, cmd = m.sidebar.Update(msg, m.sessions)
		return m, cmd
	}

	// Pass key to textarea, then check triggers
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.checkAutocompleteTriggers()
	return m, cmd
}

func (m *Model) checkAutocompleteTriggers() {
	text := m.input.Value()

	// Check for "/" at the start of input
	if strings.HasPrefix(text, "/") {
		query := text[1:]
		m.autocomplete.Show(query)
		m.mention.Hide()
		return
	}

	// Check for "@" mention trigger
	atIdx := strings.LastIndex(text, "@")
	if atIdx >= 0 {
		before := ""
		if atIdx > 0 {
			before = string(text[atIdx-1])
		}
		if atIdx == 0 || before == " " || before == "\n" {
			query := text[atIdx+1:]
			if !strings.Contains(query, " ") {
				m.mention.Show(query)
				m.autocomplete.Hide()
				return
			}
		}
	}

	// Neither trigger active
	if m.autocomplete.visible {
		m.autocomplete.Hide()
	}
	if m.mention.visible {
		m.mention.Hide()
	}
}

func (m *Model) executeSlashCommand(cmd *SlashCommand) tea.Cmd {
	switch cmd.Name {
	case "new":
		return createSession(m.store, m.workspace)
	case "sessions":
		return m.openSessionDialog()
	case "agents":
		return m.openAgentDialog()
	case "models":
		return m.openModelDialog()
	case "themes":
		return m.openThemeDialog()
	case "exit":
		return tea.Quit
	case "sidebar":
		m.toggleSidebar()
		return nil
	case "home":
		m.route = routeHome
		return nil
	case "help":
		m.dialogs.Push(NewAlertDialog("Keyboard Shortcuts",
			"ctrl+x → leader key\n"+
				"  n: new session\n"+
				"  b: sidebar\n"+
				"  a: agents\n"+
				"  l: sessions\n"+
				"  h: home\n"+
				"  q: quit\n\n"+
				"ctrl+p: command palette\n"+
				"pgup/pgdown: scroll\n"+
				"esc: cancel/back\n"+
				"/: slash commands\n"+
				"@: file/agent mentions"))
		return nil
	case "status":
		info := "Agent: " + m.agentName + "\n"
		info += "Model: " + m.modelName + "\n"
		info += "Mode: " + m.mode + "\n"
		info += "Session: " + m.session + "\n"
		info += "Messages: " + fmt.Sprintf("%d", len(m.messages)) + "\n"
		m.dialogs.Push(NewAlertDialog("Status", info))
		return nil
	}
	return nil
}

func (m *Model) openThemeDialog() tea.Cmd {
	names := ThemeNames()
	m.dialogs.Push(NewSelectDialogWithKind("Select Theme", names, dialogKindTheme))
	return nil
}

func (m *Model) openCommandPalette() {
	var items []PaletteItem

	// Session commands
	items = append(items,
		PaletteItem{Label: "New Session", Category: "Session", Action: func() tea.Cmd {
			return createSession(m.store, m.workspace)
		}},
		PaletteItem{Label: "Switch Session", Category: "Session", Action: func() tea.Cmd {
			return m.openSessionDialog()
		}},
	)

	// Agent commands
	items = append(items,
		PaletteItem{Label: "Switch Agent", Category: "Agent", Action: func() tea.Cmd {
			return m.openAgentDialog()
		}},
		PaletteItem{Label: "Switch Model", Category: "Agent", Action: func() tea.Cmd {
			return m.openModelDialog()
		}},
	)

	// Theme commands
	for _, name := range ThemeNames() {
		themeName := name
		items = append(items, PaletteItem{
			Label: "Theme: " + themeName, Category: "Appearance",
			Action: func() tea.Cmd {
				if t, ok := BuiltinThemes[themeName]; ok {
					m.theme = t
					m.chat = newChat(t, m.chat.viewport.Width, m.chat.viewport.Height)
					m.recalcLayout()
					m.chat.Refresh(m.messages, "", m.theme)
				}
				return nil
			},
		})
	}

	// Navigation
	items = append(items,
		PaletteItem{Label: "Go Home", Category: "Navigation", Action: func() tea.Cmd {
			m.route = routeHome
			return nil
		}},
		PaletteItem{Label: "Toggle Sidebar", Category: "Navigation", Action: func() tea.Cmd {
			m.toggleSidebar()
			return nil
		}},
		PaletteItem{Label: "Show Help", Category: "Help", Action: func() tea.Cmd {
			return m.executeSlashCommand(&SlashCommand{Name: "help"})
		}},
		PaletteItem{Label: "Show Status", Category: "Info", Action: func() tea.Cmd {
			return m.executeSlashCommand(&SlashCommand{Name: "status"})
		}},
		PaletteItem{Label: "Exit", Category: "Application", Action: func() tea.Cmd {
			return tea.Quit
		}},
	)

	m.palette.Open(items)
}

func (m *Model) dispatchLeader(key string) tea.Cmd {
	switch key {
	case "n":
		return createSession(m.store, m.workspace)
	case "b":
		m.toggleSidebar()
		return nil
	case "a":
		return m.openAgentDialog()
	case "l":
		return m.openSessionDialog()
	case "h":
		m.route = routeHome
		return nil
	case "q":
		return tea.Quit
	}
	return nil
}

func (m *Model) toggleSidebar() {
	if m.active == focusSidebar {
		m.active = focusChat
	} else {
		m.active = focusSidebar
	}
	m.recalcLayout()
}

func (m *Model) openAgentDialog() tea.Cmd {
	agents := runtime.ListSubagents()
	if len(agents) == 0 {
		return nil
	}
	names := make([]string, len(agents))
	for i, a := range agents {
		names[i] = a.Name
	}
	m.dialogs.Push(NewSelectDialogWithKind("Select Agent", names, dialogKindAgent))
	return nil
}

func (m *Model) openModelDialog() tea.Cmd {
	modelMap := m.engine.ListModels()
	if len(modelMap) == 0 {
		m.dialogs.Push(NewAlertDialog("No Models", "No providers are configured."))
		return nil
	}
	var items []string
	for provider, models := range modelMap {
		for _, model := range models {
			items = append(items, provider+"/"+model)
		}
	}
	m.dialogs.Push(NewSelectDialogWithKind("Select Model", items, dialogKindModel))
	return nil
}

func (m *Model) openSessionDialog() tea.Cmd {
	if len(m.sessions) == 0 {
		return nil
	}
	names := make([]string, len(m.sessions))
	for i, s := range m.sessions {
		name := s.Title
		if name == "" {
			name = s.ID
		}
		names[i] = name
	}
	m.dialogs.Push(NewSelectDialogWithKind("Select Session", names, dialogKindSession))
	return nil
}

func (m *Model) updateDialog(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.dialogs.Empty() {
		return m, nil
	}
	top := m.dialogs.Top()
	updated, cmd := top.Update(msg)
	m.dialogs.stack[len(m.dialogs.stack)-1] = updated

	if updated.Done() {
		m.dialogs.Pop()
		return m.handleDialogResult(updated)
	}
	return m, cmd
}

func (m *Model) handleDialogResult(d Dialog) (tea.Model, tea.Cmd) {
	switch dialog := d.(type) {
	case *confirmDialogWithChannel:
		_ = dialog
	case *SelectDialog:
		selected, _ := dialog.Result().(string)
		if selected == "" {
			return m, nil
		}
		switch dialog.kind {
		case dialogKindAgent:
			m.agentName = selected
		case dialogKindModel:
			m.engine.SetModel(selected)
			m.modelName = selected
			cmd := m.toasts.Add("Model: "+selected, ToastSuccess, 2*time.Second)
			return m, cmd
		case dialogKindSession:
			for _, s := range m.sessions {
				name := s.Title
				if name == "" {
					name = s.ID
				}
				if name == selected {
					m.session = s.ID
					m.route = routeSession
					return m, loadMessages(m.store, m.workspace, m.session)
				}
			}
		case dialogKindTheme:
			if t, ok := BuiltinThemes[selected]; ok {
				m.theme = t
				m.chat = newChat(t, m.chat.viewport.Width, m.chat.viewport.Height)
				m.recalcLayout()
				m.chat.Refresh(m.messages, "", m.theme)
			}
		}
	}
	return m, nil
}

func (m *Model) handleConfirmRequest(req confirmRequest) (tea.Model, tea.Cmd) {
	desc := "Tool: " + req.name
	d := &confirmDialogWithChannel{
		ConfirmDialog: *NewConfirmDialog("Permission Required", desc),
		ch:            req.ch,
	}
	m.dialogs.Push(d)
	return m, nil
}

// confirmDialogWithChannel wraps ConfirmDialog to write result to a channel when done.
type confirmDialogWithChannel struct {
	ConfirmDialog
	ch chan bool
}

func (d *confirmDialogWithChannel) Update(msg tea.Msg) (Dialog, tea.Cmd) {
	inner, cmd := d.ConfirmDialog.Update(msg)
	updated := inner.(*ConfirmDialog)
	d.ConfirmDialog = *updated
	if d.ConfirmDialog.Done() {
		d.ch <- d.ConfirmDialog.yes
	}
	return d, cmd
}

func (d *confirmDialogWithChannel) Done() bool { return d.ConfirmDialog.Done() }
func (d *confirmDialogWithChannel) Result() any { return d.ConfirmDialog.Result() }
func (d *confirmDialogWithChannel) View(w, h int, theme Theme) string {
	return d.ConfirmDialog.View(w, h, theme)
}

func (m *Model) recalcLayout() {
	sidebarW := 0
	if m.active == focusSidebar {
		sidebarW = min(30, m.width/3)
	}
	chatW := m.width - sidebarW
	headerH := 1
	footerH := 1
	inputH := 4
	chatH := m.height - headerH - footerH - inputH
	if chatH < 1 {
		chatH = 1
	}
	_ = chatW
	m.chat.SetSize(chatW, chatH)
}

func (m *Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Command palette takes priority over everything
	if m.palette.visible {
		bg := m.renderMainView()
		paletteView := m.palette.View(m.width, m.height, m.theme)
		_ = bg
		return paletteView
	}

	if !m.dialogs.Empty() {
		bg := m.renderMainView()
		overlay := m.dialogs.Top().View(m.width, m.height, m.theme)
		_ = bg
		return overlay
	}

	return m.renderMainView()
}

func (m *Model) renderMainView() string {
	if m.route == routeHome {
		return m.renderHomeView()
	}
	return m.renderSessionView()
}

func (m *Model) renderSessionView() string {
	sidebarW := 0
	sidebarContent := ""
	if m.active == focusSidebar {
		sidebarW = min(30, m.width/3)
		sidebarContent = m.sidebar.View(m.sessions, m.session, sidebarW, m.height-2)
	}
	chatW := m.width - sidebarW

	title := ""
	for _, s := range m.sessions {
		if s.ID == m.session {
			title = s.Title
			break
		}
	}

	header := RenderHeader(title, m.agentName, m.modelName, chatW, m.theme)
	chatView := m.chat.View()

	acView := m.autocomplete.View(chatW, m.theme)
	mentionView := m.mention.View(chatW, m.theme)
	inputView := m.input.View(chatW, m.theme)

	hints := "ctrl+x:leader  ctrl+p:palette  /:cmds  @:files"
	if m.leader.IsActive() {
		hints = "n:new  b:sidebar  a:agent  l:sessions  h:home  q:quit"
	}
	busyOrStream := m.busy || m.streaming
	footer := RenderFooter(m.mode, m.err, busyOrStream, m.leader.IsActive(), hints, chatW, m.theme, m.spinner)

	parts := []string{header, chatView}
	if acView != "" {
		parts = append(parts, acView)
	}
	if mentionView != "" {
		parts = append(parts, mentionView)
	}
	parts = append(parts, inputView, footer)

	// Overlay toasts at the top right
	main := lipgloss.JoinVertical(lipgloss.Left, parts...)

	if sidebarW > 0 {
		main = lipgloss.JoinHorizontal(lipgloss.Top, sidebarContent, main)
	}

	// Render toasts on top
	if m.toasts.HasToasts() {
		toastView := m.toasts.View(m.width, m.theme)
		lines := strings.Split(main, "\n")
		toastLines := strings.Split(strings.TrimRight(toastView, "\n"), "\n")
		for i, tl := range toastLines {
			if i < len(lines) {
				padW := m.width - lipgloss.Width(tl)
				if padW > 0 {
					lines[i] = strings.Repeat(" ", padW) + tl
				}
			}
		}
		main = strings.Join(lines, "\n")
	}

	return main
}

func (m *Model) renderHomeView() string {
	return RenderHome(m.width, m.height, m.agentName, m.modelName, m.workspace,
		len(m.sessions), m.input, m.theme, m.spinner, m.busy || m.streaming)
}

func defaultSlashCommands() []SlashCommand {
	return []SlashCommand{
		{Name: "new", Aliases: []string{"clear"}, Description: "Create a new session"},
		{Name: "sessions", Aliases: []string{"resume", "continue"}, Description: "List and switch sessions"},
		{Name: "agents", Description: "Switch agent"},
		{Name: "models", Description: "List available models"},
		{Name: "themes", Description: "Switch theme"},
		{Name: "help", Description: "Show help"},
		{Name: "exit", Aliases: []string{"quit", "q"}, Description: "Exit the application"},
		{Name: "compact", Aliases: []string{"summarize"}, Description: "Compact current session"},
		{Name: "undo", Description: "Undo last user message"},
		{Name: "redo", Description: "Redo undone message"},
		{Name: "export", Description: "Export session transcript"},
		{Name: "rename", Description: "Rename current session"},
		{Name: "status", Description: "Show status information"},
		{Name: "sidebar", Description: "Toggle sidebar"},
		{Name: "home", Description: "Go to home screen"},
	}
}
