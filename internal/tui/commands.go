package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/morefun2602/opencode-go/internal/runtime"
	"github.com/morefun2602/opencode-go/internal/store"
)

// --- Message types ---

type sessionsLoaded struct{ sessions []store.SessionRow }
type messagesLoaded struct{ messages []store.MessageRow }
type sessionCreated struct{ id string }
type sessionSelected struct{ id string }
type turnComplete struct {
	reply string
	err   error
}

type streamStarted struct{ cancel context.CancelFunc }
type streamChunk struct{ text string }
type streamDone struct{ err error }

type confirmRequest struct {
	name string
	args map[string]any
	ch   chan bool
}

// NewConfirmRequest creates a confirmRequest message to be sent via p.Send().
func NewConfirmRequest(name string, args map[string]any, ch chan bool) confirmRequest {
	return confirmRequest{name: name, args: args, ch: ch}
}

// --- Commands ---

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

// streamMessage launches a goroutine that uses CompleteTurnStream, sending
// streamChunk messages back to the TUI via p.Send(). This is the recommended
// Bubble Tea pattern for long-running operations with multiple callbacks.
func streamMessage(p *tea.Program, eng *runtime.Engine, ws, session, text string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		p.Send(streamStarted{cancel: cancel})

		err := eng.CompleteTurnStream(ctx, ws, session, text, func(chunk string) error {
			p.Send(streamChunk{text: chunk})
			return nil
		})
		return streamDone{err: err}
	}
}

// createSessionAndSend creates a new session and immediately sends the first
// message on it. Used when the user types from the Home screen with no session.
func createSessionAndSend(st store.Store, eng *runtime.Engine, ws, text string, p *tea.Program) tea.Cmd {
	return func() tea.Msg {
		id, err := st.CreateSession(context.Background(), ws)
		if err != nil {
			return turnComplete{err: err}
		}
		if p != nil {
			p.Send(sessionCreated{id: id})
			ctx, cancel := context.WithCancel(context.Background())
			p.Send(streamStarted{cancel: cancel})
			streamErr := eng.CompleteTurnStream(ctx, ws, id, text, func(chunk string) error {
				p.Send(streamChunk{text: chunk})
				return nil
			})
			return streamDone{err: streamErr}
		}
		reply, sendErr := eng.CompleteTurn(context.Background(), ws, id, text)
		return turnComplete{reply: reply, err: sendErr}
	}
}
