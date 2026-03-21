package tui

import (
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

// Model is the top-level Bubble Tea model for the TUI application.
type Model struct {
	engine    *runtime.Engine
	store     store.Store
	workspace string
	theme     Theme
	width     int
	height    int
	active    focus
	mode      string

	chat     chatModel
	sidebar  sidebarModel
	input    inputModel
	sessions []store.SessionRow
	session  string
	messages []store.MessageRow
	busy     bool
	err      error
}

// New creates a new top-level TUI model.
func New(eng *runtime.Engine, st store.Store, workspace string, theme Theme) Model {
	return Model{
		engine:    eng,
		store:     st,
		workspace: workspace,
		theme:     theme,
		mode:      "build",
		active:    focusChat,
		chat:      newChat(theme),
		sidebar:   newSidebar(theme),
		input:     newInput(theme),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		loadSessions(m.store, m.workspace),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case sessionsLoaded:
		m.sessions = msg.sessions
		if len(m.sessions) > 0 {
			m.session = m.sessions[0].ID
			return m, loadMessages(m.store, m.workspace, m.session)
		}
		return m, createSession(m.store, m.workspace)

	case sessionCreated:
		m.session = msg.id
		return m, loadSessions(m.store, m.workspace)

	case messagesLoaded:
		m.messages = msg.messages
		return m, nil

	case turnComplete:
		m.busy = false
		m.err = msg.err
		return m, loadMessages(m.store, m.workspace, m.session)

	case sessionSelected:
		m.session = msg.id
		return m, loadMessages(m.store, m.workspace, m.session)
	}

	if m.active == focusSidebar {
		var cmd tea.Cmd
		m.sidebar, cmd = m.sidebar.Update(msg, m.sessions)
		return m, cmd
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "ctrl+p":
		switch m.mode {
		case "build":
			m.mode = "plan"
		case "plan":
			m.mode = "explore"
		default:
			m.mode = "build"
		}
		return m, nil
	case "ctrl+b":
		if m.active == focusSidebar {
			m.active = focusChat
		} else {
			m.active = focusSidebar
		}
		return m, nil
	case "ctrl+n":
		return m, createSession(m.store, m.workspace)
	}

	if m.active == focusChat && msg.String() == "enter" && !m.busy {
		text := m.input.Value()
		if text == "" {
			return m, nil
		}
		m.input.Reset()
		m.busy = true
		return m, sendMessage(m.engine, m.workspace, m.session, text)
	}

	if m.active == focusSidebar {
		var cmd tea.Cmd
		m.sidebar, cmd = m.sidebar.Update(msg, m.sessions)
		return m, cmd
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	sidebar := ""
	sidebarW := 0
	if m.active == focusSidebar {
		sidebarW = min(30, m.width/3)
		sidebar = m.sidebar.View(m.sessions, m.session, sidebarW, m.height-2)
	}

	chatW := m.width - sidebarW
	statusBar := m.statusBar(chatW)
	inputH := 4
	chatH := m.height - inputH - 1

	chatContent := m.chat.View(m.messages, chatW, chatH, m.theme)
	inputContent := m.input.View(chatW, m.theme)

	main := lipgloss.JoinVertical(lipgloss.Left, chatContent, inputContent, statusBar)
	if sidebarW > 0 {
		return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)
	}
	return main
}

func (m Model) statusBar(w int) string {
	modeStyle := lipgloss.NewStyle().
		Foreground(m.theme.Background).
		Background(m.theme.Primary).
		Padding(0, 1)

	sessionStyle := lipgloss.NewStyle().
		Foreground(m.theme.Subtle)

	hintStyle := lipgloss.NewStyle().
		Foreground(m.theme.Subtle)

	modeText := modeStyle.Render(m.mode)

	sessionText := sessionStyle.Render(" " + truncate(m.session, 12))

	busyText := ""
	if m.busy {
		busyText = lipgloss.NewStyle().
			Foreground(m.theme.Success).
			Render(" ⟳ thinking...")
	}

	errText := ""
	if m.err != nil {
		errText = lipgloss.NewStyle().
			Foreground(m.theme.Error).
			Render(" err: " + truncate(m.err.Error(), 30))
	}

	hint := hintStyle.Render("  ctrl+p:mode ctrl+b:sidebar ctrl+n:new ctrl+c:quit")

	return lipgloss.NewStyle().
		Width(w).
		Background(m.theme.Border).
		Render(modeText + sessionText + busyText + errText + hint)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
