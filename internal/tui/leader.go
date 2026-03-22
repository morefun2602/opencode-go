package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type leaderTimeout struct{}

// LeaderState tracks the leader key two-phase shortcut state.
type LeaderState struct {
	active  bool
	timeout time.Duration
}

func NewLeaderState() LeaderState {
	return LeaderState{timeout: 1500 * time.Millisecond}
}

func (l *LeaderState) Activate() tea.Cmd {
	l.active = true
	return tea.Tick(l.timeout, func(time.Time) tea.Msg {
		return leaderTimeout{}
	})
}

func (l *LeaderState) Deactivate() {
	l.active = false
}

func (l *LeaderState) IsActive() bool {
	return l.active
}
