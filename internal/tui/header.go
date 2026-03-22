package tui

import "github.com/charmbracelet/lipgloss"

// RenderHeader renders the top header bar showing session title, agent, and model.
func RenderHeader(title, agent, model string, w int, theme Theme) string {
	if title == "" {
		title = "New Session"
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Text)

	rightStyle := lipgloss.NewStyle().
		Foreground(theme.Subtle)

	right := ""
	if agent != "" {
		right += agent
	}
	if model != "" {
		if right != "" {
			right += " │ "
		}
		right += model
	}

	titleW := w - lipgloss.Width(right) - 2
	if titleW < 10 {
		titleW = 10
	}

	left := titleStyle.Width(titleW).Render(truncateStr(title, titleW))
	rightRendered := rightStyle.Render(right)

	return lipgloss.NewStyle().
		Width(w).
		Background(theme.HeaderBg).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, " "+left, rightRendered+" "))
}
