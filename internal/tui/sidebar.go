package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/morefun2602/opencode-go/internal/store"
)

type sidebarModel struct {
	cursor int
	theme  Theme
}

func newSidebar(theme Theme) sidebarModel {
	return sidebarModel{theme: theme}
}

func (s sidebarModel) Update(msg tea.Msg, sessions []store.SessionRow) (sidebarModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if s.cursor > 0 {
				s.cursor--
			}
		case "down", "j":
			if s.cursor < len(sessions)-1 {
				s.cursor++
			}
		case "enter":
			if s.cursor < len(sessions) {
				return s, func() tea.Msg {
					return sessionSelected{id: sessions[s.cursor].ID}
				}
			}
		}
	}
	return s, nil
}

func (s sidebarModel) View(sessions []store.SessionRow, current string, w, h int) string {
	title := lipgloss.NewStyle().
		Foreground(s.theme.Primary).
		Bold(true).
		Padding(0, 1).
		Render("Sessions")

	var rows []string
	rows = append(rows, title)
	rows = append(rows, strings.Repeat("─", w))

	for i, sess := range sessions {
		label := sess.Title
		if label == "" {
			label = truncateStr(sess.ID, w-4)
		}
		if label == "" {
			label = fmt.Sprintf("session-%d", i+1)
		}

		ts := ""
		if sess.CreatedAt > 0 {
			ts = time.Unix(sess.CreatedAt, 0).Format("01/02 15:04")
		}

		line := label
		if ts != "" {
			pad := w - len(label) - len(ts) - 2
			if pad < 1 {
				pad = 1
			}
			line = label + strings.Repeat(" ", pad) + ts
		}

		style := lipgloss.NewStyle().Width(w).Padding(0, 1)
		if sess.ID == current {
			style = style.Foreground(s.theme.Primary).Bold(true)
		} else if i == s.cursor {
			style = style.Foreground(s.theme.Text).Background(s.theme.Border)
		} else {
			style = style.Foreground(s.theme.Subtle)
		}
		rows = append(rows, style.Render(line))
	}

	content := strings.Join(rows, "\n")

	return lipgloss.NewStyle().
		Width(w).
		Height(h).
		BorderRight(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(s.theme.Border).
		Render(content)
}
