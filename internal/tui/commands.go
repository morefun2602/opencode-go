package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/morefun2602/opencode-go/internal/runtime"
	"github.com/morefun2602/opencode-go/internal/store"
)

type sessionsLoaded struct{ sessions []store.SessionRow }
type messagesLoaded struct{ messages []store.MessageRow }
type sessionCreated struct{ id string }
type sessionSelected struct{ id string }
type turnComplete struct {
	reply string
	err   error
}

func loadSessions(st store.Store, ws string) tea.Cmd {
	return func() tea.Msg {
		rows, _ := st.ListSessions(context.Background(), ws, 50)
		return sessionsLoaded{sessions: rows}
	}
}

func loadMessages(st store.Store, ws, session string) tea.Cmd {
	return func() tea.Msg {
		rows, _ := st.ListMessages(context.Background(), ws, session, 0, 1000)
		return messagesLoaded{messages: rows}
	}
}

func createSession(st store.Store, ws string) tea.Cmd {
	return func() tea.Msg {
		id, _ := st.CreateSession(context.Background(), ws)
		return sessionCreated{id: id}
	}
}

func sendMessage(eng *runtime.Engine, ws, session, text string) tea.Cmd {
	return func() tea.Msg {
		reply, err := eng.CompleteTurn(context.Background(), ws, session, text)
		return turnComplete{reply: reply, err: err}
	}
}
