package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderDiff renders a unified diff string with theme-colored lines.
func RenderDiff(diff string, w int, theme Theme) string {
	if diff == "" {
		return ""
	}

	lines := strings.Split(diff, "\n")
	var rendered []string

	for _, line := range lines {
		var style lipgloss.Style
		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
			style = lipgloss.NewStyle().Foreground(theme.DiffHunkHeader).Bold(true)
		case strings.HasPrefix(line, "@@"):
			style = lipgloss.NewStyle().Foreground(theme.DiffHunkHeader).Italic(true)
		case strings.HasPrefix(line, "+"):
			style = lipgloss.NewStyle().
				Foreground(theme.DiffAdded).
				Background(theme.DiffAddedBg)
		case strings.HasPrefix(line, "-"):
			style = lipgloss.NewStyle().
				Foreground(theme.DiffRemoved).
				Background(theme.DiffRemovedBg)
		default:
			style = lipgloss.NewStyle().Foreground(theme.DiffContext)
		}
		rendered = append(rendered, style.Width(w-2).Render(line))
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.BorderSubtle).
		Width(w).
		Render(strings.Join(rendered, "\n"))
}

// IsDiffContent returns true if text looks like a unified diff.
func IsDiffContent(text string) bool {
	lines := strings.SplitN(text, "\n", 10)
	diffMarkers := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") ||
			strings.HasPrefix(line, "@@") || strings.HasPrefix(line, "diff --git") {
			diffMarkers++
		}
	}
	return diffMarkers >= 2
}
