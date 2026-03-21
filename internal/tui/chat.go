package tui

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/morefun2602/opencode-go/internal/store"
)

type chatModel struct {
	scroll   int
	renderer *glamour.TermRenderer
}

func newChat(theme Theme) chatModel {
	style := "dark"
	if theme.Name == "light" {
		style = "light"
	}
	r, _ := glamour.NewTermRenderer(glamour.WithStylePath(style), glamour.WithWordWrap(80))
	return chatModel{renderer: r}
}

func (c chatModel) View(msgs []store.MessageRow, w, h int, theme Theme) string {
	var lines []string
	for _, m := range msgs {
		var prefix string
		var style lipgloss.Style
		switch m.Role {
		case "user":
			prefix = "You"
			style = lipgloss.NewStyle().Foreground(theme.Primary).Bold(true)
		case "assistant":
			prefix = "Assistant"
			style = lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true)
		case "tool":
			prefix = "Tool"
			style = lipgloss.NewStyle().Foreground(theme.Subtle).Italic(true)
		default:
			continue
		}
		header := style.Render(prefix)
		body := m.Body
		if m.Role == "assistant" && c.renderer != nil {
			if rendered, err := c.renderer.Render(body); err == nil {
				body = strings.TrimSpace(rendered)
			}
		}
		if len(body) > 500 {
			body = body[:500] + "…"
		}
		lines = append(lines, header+"\n"+body+"\n")
	}
	content := strings.Join(lines, "\n")

	return lipgloss.NewStyle().
		Width(w).
		Height(h).
		Render(content)
}
