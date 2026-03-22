package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PaletteItem represents an item in the command palette.
type PaletteItem struct {
	Label    string
	Category string
	Action   func() tea.Cmd
}

// commandPalette is a Ctrl+P style overlay for discovering and executing commands.
type commandPalette struct {
	visible bool
	items   []PaletteItem
	filter  []PaletteItem
	cursor  int
	query   string
	input   strings.Builder
}

func newCommandPalette() commandPalette {
	return commandPalette{}
}

// Open shows the palette with the given items.
func (cp *commandPalette) Open(items []PaletteItem) {
	cp.visible = true
	cp.items = items
	cp.query = ""
	cp.cursor = 0
	cp.input.Reset()
	cp.refilter()
}

// Close hides the palette.
func (cp *commandPalette) Close() {
	cp.visible = false
	cp.cursor = 0
	cp.query = ""
	cp.input.Reset()
	cp.filter = nil
}

func (cp *commandPalette) refilter() {
	q := strings.ToLower(cp.query)
	cp.filter = nil
	for _, item := range cp.items {
		if q == "" || fuzzyMatch(item.Label, q) || fuzzyMatch(item.Category, q) {
			cp.filter = append(cp.filter, item)
		}
	}
	if cp.cursor >= len(cp.filter) {
		cp.cursor = max(0, len(cp.filter)-1)
	}
}

// Update processes a key message and returns a selected item action or nil.
func (cp *commandPalette) Update(msg tea.KeyMsg) (action func() tea.Cmd, consumed bool) {
	if !cp.visible {
		return nil, false
	}

	switch msg.String() {
	case "esc":
		cp.Close()
		return nil, true

	case "up", "ctrl+p":
		if cp.cursor > 0 {
			cp.cursor--
		}
		return nil, true

	case "down", "ctrl+n":
		if cp.cursor < len(cp.filter)-1 {
			cp.cursor++
		}
		return nil, true

	case "enter":
		if cp.cursor < len(cp.filter) {
			item := cp.filter[cp.cursor]
			cp.Close()
			return item.Action, true
		}
		return nil, true

	case "backspace":
		if len(cp.query) > 0 {
			cp.query = cp.query[:len(cp.query)-1]
			cp.refilter()
		}
		return nil, true

	default:
		k := msg.String()
		if len(k) == 1 && k[0] >= ' ' {
			cp.query += k
			cp.refilter()
		}
		return nil, true
	}
}

// View renders the command palette overlay.
func (cp *commandPalette) View(w, h int, theme Theme) string {
	if !cp.visible {
		return ""
	}

	paletteW := min(70, w-6)
	if paletteW < 30 {
		paletteW = 30
	}

	inputStyle := lipgloss.NewStyle().
		Foreground(theme.Text).
		Background(theme.BackgroundElem).
		Width(paletteW - 4).
		Padding(0, 1)

	queryDisplay := cp.query
	if queryDisplay == "" {
		queryDisplay = lipgloss.NewStyle().Foreground(theme.Subtle).Render("Type to search commands...")
	} else {
		queryDisplay = lipgloss.NewStyle().Foreground(theme.Text).Render("> " + queryDisplay)
	}
	inputBox := inputStyle.Render(queryDisplay)

	maxShow := min(12, len(cp.filter))
	start := 0
	if cp.cursor >= maxShow {
		start = cp.cursor - maxShow + 1
	}
	end := start + maxShow
	if end > len(cp.filter) {
		end = len(cp.filter)
	}

	var rows []string
	for i := start; i < end; i++ {
		item := cp.filter[i]
		label := item.Label
		cat := item.Category

		nameStyle := lipgloss.NewStyle()
		catStyle := lipgloss.NewStyle().Foreground(theme.Subtle)
		rowStyle := lipgloss.NewStyle().Width(paletteW - 4).Padding(0, 1)

		if i == cp.cursor {
			rowStyle = rowStyle.Background(theme.BorderActive)
			nameStyle = nameStyle.Foreground(theme.Text).Bold(true)
			catStyle = catStyle.Foreground(theme.Subtle)
		} else {
			nameStyle = nameStyle.Foreground(theme.Text)
		}

		catRender := ""
		if cat != "" {
			catRender = catStyle.Render("  [" + cat + "]")
		}

		row := rowStyle.Render(nameStyle.Render(label) + catRender)
		rows = append(rows, row)
	}

	var content string
	if len(rows) == 0 {
		noResult := lipgloss.NewStyle().
			Foreground(theme.Subtle).
			Width(paletteW - 4).
			Padding(0, 1).
			Align(lipgloss.Center).
			Render("No matching commands")
		content = inputBox + "\n" + lipgloss.NewStyle().Foreground(theme.Border).Render(strings.Repeat("─", paletteW-4)) + "\n" + noResult
	} else {
		content = inputBox + "\n" + lipgloss.NewStyle().Foreground(theme.Border).Render(strings.Repeat("─", paletteW-4)) + "\n" + strings.Join(rows, "\n")
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Primary).
		Padding(1, 1).
		Width(paletteW).
		Render(content)

	return lipgloss.Place(w, h/3, lipgloss.Center, lipgloss.Top, box,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(theme.DialogOverlay))
}
