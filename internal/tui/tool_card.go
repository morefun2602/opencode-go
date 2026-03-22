package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type toolRenderer func(args map[string]any, result string, isErr bool, w int, theme Theme) string

var toolRenderers = map[string]toolRenderer{
	"bash":  renderBash,
	"read":  renderRead,
	"edit":  renderEdit,
	"write": renderWrite,
	"grep":  renderGrep,
	"glob":  renderGlob,
}

// RenderToolCard renders a structured card for a tool call with optional result.
func RenderToolCard(name string, args map[string]any, result string, isErr bool, w int, theme Theme) string {
	if w < 10 {
		w = 40
	}
	innerW := w - 4

	statusIcon := "✓"
	statusColor := theme.Success
	if result == "" {
		statusIcon = "⟳"
		statusColor = theme.Secondary
	} else if isErr {
		statusIcon = "✗"
		statusColor = theme.Error
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(theme.Text).
		Background(theme.ToolHeader).
		Width(innerW).
		Padding(0, 1)

	header := headerStyle.Render(
		lipgloss.NewStyle().Foreground(statusColor).Render(statusIcon) +
			" " +
			lipgloss.NewStyle().Bold(true).Render(name),
	)

	var body string
	if fn, ok := toolRenderers[name]; ok {
		body = fn(args, result, isErr, innerW, theme)
	} else {
		body = renderGeneric(args, result, isErr, innerW, theme)
	}

	content := header
	if body != "" {
		content = content + "\n" + body
	}

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ToolBorder).
		Width(innerW + 2).
		Padding(0, 0)

	return border.Render(content)
}

func renderBash(args map[string]any, result string, isErr bool, w int, theme Theme) string {
	cmd, _ := args["command"].(string)
	if cmd == "" {
		cmd, _ = args["cmd"].(string)
	}
	cmdLine := lipgloss.NewStyle().Foreground(theme.Subtle).Padding(0, 1).Render(
		"$ " + truncateStr(cmd, w-4),
	)
	if result == "" {
		return cmdLine
	}
	return cmdLine + "\n" + foldResult(result, w, theme)
}

func renderRead(args map[string]any, result string, isErr bool, w int, theme Theme) string {
	path, _ := args["path"].(string)
	info := lipgloss.NewStyle().Foreground(theme.Subtle).Padding(0, 1).Render(path)
	if result != "" && !isErr {
		lines := strings.Count(result, "\n") + 1
		info += lipgloss.NewStyle().Foreground(theme.Subtle).Render(
			fmt.Sprintf("  (%d lines)", lines),
		)
	}
	return info
}

func renderEdit(args map[string]any, result string, isErr bool, w int, theme Theme) string {
	path, _ := args["path"].(string)
	if path == "" {
		path, _ = args["file_path"].(string)
	}
	return lipgloss.NewStyle().Foreground(theme.Subtle).Padding(0, 1).Render(path)
}

func renderWrite(args map[string]any, result string, isErr bool, w int, theme Theme) string {
	path, _ := args["path"].(string)
	if path == "" {
		path, _ = args["file_path"].(string)
	}
	return lipgloss.NewStyle().Foreground(theme.Subtle).Padding(0, 1).Render(path)
}

func renderGrep(args map[string]any, result string, isErr bool, w int, theme Theme) string {
	pattern, _ := args["pattern"].(string)
	info := lipgloss.NewStyle().Foreground(theme.Subtle).Padding(0, 1).Render(
		"/" + truncateStr(pattern, w-6) + "/",
	)
	if result != "" && !isErr {
		matches := strings.Count(result, "\n") + 1
		info += lipgloss.NewStyle().Foreground(theme.Subtle).Render(
			fmt.Sprintf("  (%d matches)", matches),
		)
	}
	return info
}

func renderGlob(args map[string]any, result string, isErr bool, w int, theme Theme) string {
	pattern, _ := args["pattern"].(string)
	if pattern == "" {
		pattern, _ = args["glob_pattern"].(string)
	}
	info := lipgloss.NewStyle().Foreground(theme.Subtle).Padding(0, 1).Render(
		truncateStr(pattern, w-4),
	)
	if result != "" && !isErr {
		matches := strings.Count(result, "\n") + 1
		info += lipgloss.NewStyle().Foreground(theme.Subtle).Render(
			fmt.Sprintf("  (%d files)", matches),
		)
	}
	return info
}

func renderGeneric(args map[string]any, result string, isErr bool, w int, theme Theme) string {
	summary := ""
	if len(args) > 0 {
		b, _ := json.Marshal(args)
		summary = string(b)
	}
	return lipgloss.NewStyle().Foreground(theme.Subtle).Padding(0, 1).Render(
		truncateStr(summary, w-4),
	)
}

func foldResult(result string, w int, theme Theme) string {
	if IsDiffContent(result) {
		return RenderDiff(result, w-2, theme)
	}

	lines := strings.Split(result, "\n")
	style := lipgloss.NewStyle().Foreground(theme.Subtle).Padding(0, 1)
	if len(lines) <= 5 {
		return style.Render(result)
	}
	shown := strings.Join(lines[:3], "\n")
	more := fmt.Sprintf("▸ %d more lines", len(lines)-3)
	return style.Render(shown) + "\n" + lipgloss.NewStyle().Foreground(theme.Subtle).Italic(true).Padding(0, 1).Render(more)
}

func truncateStr(s string, n int) string {
	if n < 3 {
		n = 3
	}
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
