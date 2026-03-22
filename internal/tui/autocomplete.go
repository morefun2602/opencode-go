package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SlashCommand defines a TUI slash command.
type SlashCommand struct {
	Name        string
	Aliases     []string
	Description string
	Action      func() tea.Cmd
}

// autocompleteModel manages the "/" command popup.
type autocompleteModel struct {
	visible  bool
	commands []SlashCommand
	filtered []SlashCommand
	cursor   int
	query    string
}

func newAutocomplete(commands []SlashCommand) autocompleteModel {
	return autocompleteModel{commands: commands}
}

// Show opens the autocomplete popup with the given query.
func (a *autocompleteModel) Show(query string) {
	a.visible = true
	a.query = query
	a.filter()
}

// Hide closes the popup.
func (a *autocompleteModel) Hide() {
	a.visible = false
	a.cursor = 0
	a.query = ""
	a.filtered = nil
}

func (a *autocompleteModel) filter() {
	q := strings.ToLower(a.query)
	a.filtered = nil
	for _, cmd := range a.commands {
		if q == "" {
			a.filtered = append(a.filtered, cmd)
			continue
		}
		if fuzzyMatch(cmd.Name, q) || fuzzyMatch(cmd.Description, q) {
			a.filtered = append(a.filtered, cmd)
			continue
		}
		for _, alias := range cmd.Aliases {
			if fuzzyMatch(alias, q) {
				a.filtered = append(a.filtered, cmd)
				break
			}
		}
	}
	if a.cursor >= len(a.filtered) {
		a.cursor = max(0, len(a.filtered)-1)
	}
}

func fuzzyMatch(text, query string) bool {
	text = strings.ToLower(text)
	qi := 0
	for i := 0; i < len(text) && qi < len(query); i++ {
		if text[i] == query[qi] {
			qi++
		}
	}
	return qi == len(query)
}

// Update handles keyboard navigation for the popup.
func (a *autocompleteModel) Update(msg tea.KeyMsg) (selected *SlashCommand, consumed bool) {
	if !a.visible {
		return nil, false
	}
	switch msg.String() {
	case "up", "ctrl+p":
		if a.cursor > 0 {
			a.cursor--
		}
		return nil, true
	case "down", "ctrl+n":
		if a.cursor < len(a.filtered)-1 {
			a.cursor++
		}
		return nil, true
	case "enter", "tab":
		if a.cursor < len(a.filtered) {
			sel := a.filtered[a.cursor]
			a.Hide()
			return &sel, true
		}
		return nil, true
	case "esc":
		a.Hide()
		return nil, true
	}
	return nil, false
}

// View renders the popup (positioned above the input area).
func (a *autocompleteModel) View(w int, theme Theme) string {
	if !a.visible || len(a.filtered) == 0 {
		return ""
	}

	maxShow := 8
	items := a.filtered
	if len(items) > maxShow {
		items = items[:maxShow]
	}

	popupW := w - 4
	if popupW > 60 {
		popupW = 60
	}
	if popupW < 20 {
		popupW = 20
	}

	var rows []string
	for i, cmd := range items {
		name := "/" + cmd.Name
		desc := cmd.Description
		nameStyle := lipgloss.NewStyle().Bold(true)
		descStyle := lipgloss.NewStyle().Foreground(theme.Subtle)

		if i == a.cursor {
			nameStyle = nameStyle.Foreground(theme.Text).Background(theme.BorderActive)
			descStyle = descStyle.Background(theme.BorderActive)
		} else {
			nameStyle = nameStyle.Foreground(theme.Primary)
		}

		nameW := lipgloss.Width(name) + 1
		descW := popupW - nameW - 4
		if descW < 0 {
			descW = 0
		}
		descRendered := ""
		if descW > 3 {
			descRendered = descStyle.Render(" " + truncateStr(desc, descW))
		}

		row := nameStyle.Render(name) + descRendered
		if i == a.cursor {
			row = lipgloss.NewStyle().Width(popupW - 2).Background(theme.BorderActive).Render(row)
		}
		rows = append(rows, row)
	}

	content := strings.Join(rows, "\n")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Border).
		Padding(0, 1).
		Width(popupW).
		Render(content)
}
