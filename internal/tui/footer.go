package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// RenderFooter renders the bottom status bar with mode, status, and key hints.
func RenderFooter(mode string, errVal error, busy bool, leaderActive bool, hints string, w int, theme Theme, sp spinner.Model) string {
	modeStyle := lipgloss.NewStyle().
		Foreground(theme.Background).
		Background(theme.Primary).
		Padding(0, 1)
	modeText := modeStyle.Render(mode)

	var mid string
	if leaderActive {
		mid = lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true).
			Render(" -- LEADER -- ")
	} else if errVal != nil {
		mid = lipgloss.NewStyle().
			Foreground(theme.Error).
			Render(" " + truncateStr(errVal.Error(), 40))
	} else if busy {
		mid = lipgloss.NewStyle().
			Foreground(theme.Success).
			Render(" " + sp.View() + " thinking...")
	}

	hintStyle := lipgloss.NewStyle().Foreground(theme.Subtle)
	hintText := hintStyle.Render(hints)

	midW := w - lipgloss.Width(modeText) - lipgloss.Width(hintText) - 1
	if midW < 0 {
		midW = 0
	}
	midRendered := lipgloss.NewStyle().Width(midW).Render(mid)

	return lipgloss.NewStyle().
		Width(w).
		Background(theme.FooterBg).
		Render(modeText + midRendered + hintText)
}
