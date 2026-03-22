package tui

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

var tips = []string{
	"Press {ctrl+x} then {n} to create a new session",
	"Press {ctrl+x} then {b} to toggle the sidebar",
	"Press {ctrl+x} then {a} to switch agents",
	"Press {ctrl+x} then {l} to list sessions",
	"Press {esc} to cancel the current generation",
	"Press {pgup}/{pgdown} to scroll the chat",
	"Type a message and press {enter} to start chatting",
}

// RenderHome renders the home/welcome screen.
func RenderHome(w, h int, agent, model, workspace string, sessionCount int, input inputModel, theme Theme, sp spinner.Model, busy bool) string {
	if w < 10 || h < 10 {
		return ""
	}

	logo := RenderLogo(theme)
	logoLines := strings.Count(logo, "\n") + 1

	promptW := w - 4
	if promptW > 75 {
		promptW = 75
	}

	subtitle := lipgloss.NewStyle().
		Foreground(theme.Subtle).
		Render("AI-powered coding assistant")

	inputView := input.View(promptW, theme)

	tip := renderTip(theme)

	statusParts := []string{}
	if workspace != "" {
		statusParts = append(statusParts, lipgloss.NewStyle().Foreground(theme.Subtle).Render(truncateStr(workspace, 40)))
	}
	if agent != "" {
		statusParts = append(statusParts,
			lipgloss.NewStyle().Foreground(theme.Primary).Render(agent))
	}
	if model != "" {
		statusParts = append(statusParts,
			lipgloss.NewStyle().Foreground(theme.Subtle).Render(model))
	}
	if sessionCount > 0 {
		statusParts = append(statusParts,
			lipgloss.NewStyle().Foreground(theme.Subtle).Render(fmt.Sprintf("%d sessions", sessionCount)))
	}
	status := strings.Join(statusParts, lipgloss.NewStyle().Foreground(theme.Border).Render(" │ "))

	center := lipgloss.NewStyle().Width(w).Align(lipgloss.Center)

	content := strings.Join([]string{
		center.Render(logo),
		"",
		center.Render(subtitle),
		"",
		center.Render(inputView),
		"",
		"",
		center.Render(tip),
	}, "\n")

	contentH := logoLines + 2 + 1 + 4 + 3 + 2
	topPad := (h - contentH - 3) / 3
	if topPad < 1 {
		topPad = 1
	}

	footer := lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(status)

	var busyLine string
	if busy {
		busyLine = lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(
			sp.View() + " thinking...")
	}

	var sb strings.Builder
	sb.WriteString(strings.Repeat("\n", topPad))
	sb.WriteString(content)
	sb.WriteString("\n")
	if busyLine != "" {
		sb.WriteString("\n")
		sb.WriteString(busyLine)
	}

	currentH := topPad + contentH
	if busy {
		currentH += 2
	}
	bottomPad := h - currentH - 2
	if bottomPad < 0 {
		bottomPad = 0
	}
	sb.WriteString(strings.Repeat("\n", bottomPad))
	sb.WriteString(footer)

	return sb.String()
}

func renderTip(theme Theme) string {
	tip := tips[rand.Intn(len(tips))]

	var sb strings.Builder
	normal := lipgloss.NewStyle().Foreground(theme.Subtle)
	key := lipgloss.NewStyle().Foreground(theme.Primary).Bold(true)

	i := 0
	for i < len(tip) {
		if tip[i] == '{' {
			end := strings.IndexByte(tip[i:], '}')
			if end >= 0 {
				sb.WriteString(key.Render(tip[i+1 : i+end]))
				i += end + 1
				continue
			}
		}
		sb.WriteByte(tip[i])
		i++
	}

	return normal.Render("💡 ") + sb.String()
}
