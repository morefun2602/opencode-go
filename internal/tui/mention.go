package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MentionKind categorizes the type of @ mention.
type MentionKind int

const (
	MentionFile MentionKind = iota
	MentionAgent
)

// MentionItem represents a single mention suggestion.
type MentionItem struct {
	Label string
	Value string
	Kind  MentionKind
}

// mentionModel manages the "@" mention autocomplete popup.
type mentionModel struct {
	visible   bool
	items     []MentionItem
	filtered  []MentionItem
	cursor    int
	query     string
	workspace string
}

func newMentionModel(workspace string) mentionModel {
	return mentionModel{workspace: workspace}
}

// Show opens the mention popup with a query.
func (mm *mentionModel) Show(query string) {
	mm.visible = true
	mm.query = query
	mm.loadAndFilter()
}

// Hide closes the popup.
func (mm *mentionModel) Hide() {
	mm.visible = false
	mm.cursor = 0
	mm.query = ""
	mm.filtered = nil
}

func (mm *mentionModel) loadAndFilter() {
	if mm.items == nil {
		mm.items = mm.scanFiles()
	}

	q := strings.ToLower(mm.query)
	mm.filtered = nil
	for _, item := range mm.items {
		if q == "" || fuzzyMatch(item.Label, q) {
			mm.filtered = append(mm.filtered, item)
		}
	}
	if len(mm.filtered) > 20 {
		mm.filtered = mm.filtered[:20]
	}
	if mm.cursor >= len(mm.filtered) {
		mm.cursor = max(0, len(mm.filtered)-1)
	}
}

func (mm *mentionModel) scanFiles() []MentionItem {
	var items []MentionItem
	root := mm.workspace
	if root == "" {
		root = "."
	}

	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if name == ".git" || name == "node_modules" || name == ".next" || name == "__pycache__" || name == "vendor" {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if rel != "" {
			items = append(items, MentionItem{
				Label: rel,
				Value: rel,
				Kind:  MentionFile,
			})
		}
		if len(items) > 200 {
			return filepath.SkipAll
		}
		return nil
	})

	sort.Slice(items, func(i, j int) bool {
		return items[i].Label < items[j].Label
	})
	return items
}

// Update handles navigation keys for the mention popup.
func (mm *mentionModel) Update(msg tea.KeyMsg) (selected *MentionItem, consumed bool) {
	if !mm.visible {
		return nil, false
	}
	switch msg.String() {
	case "up", "ctrl+p":
		if mm.cursor > 0 {
			mm.cursor--
		}
		return nil, true
	case "down", "ctrl+n":
		if mm.cursor < len(mm.filtered)-1 {
			mm.cursor++
		}
		return nil, true
	case "enter", "tab":
		if mm.cursor < len(mm.filtered) {
			sel := mm.filtered[mm.cursor]
			mm.Hide()
			return &sel, true
		}
		return nil, true
	case "esc":
		mm.Hide()
		return nil, true
	}
	return nil, false
}

// View renders the mention popup.
func (mm *mentionModel) View(w int, theme Theme) string {
	if !mm.visible || len(mm.filtered) == 0 {
		return ""
	}

	maxShow := 8
	items := mm.filtered
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
	for i, item := range items {
		icon := "📄"
		if item.Kind == MentionAgent {
			icon = "🤖"
		}
		label := icon + " " + item.Label

		nameStyle := lipgloss.NewStyle()
		if i == mm.cursor {
			nameStyle = nameStyle.Foreground(theme.Text).Background(theme.BorderActive).Width(popupW - 2)
		} else {
			nameStyle = nameStyle.Foreground(theme.Subtle)
		}
		rows = append(rows, nameStyle.Render(label))
	}

	content := strings.Join(rows, "\n")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Border).
		Padding(0, 1).
		Width(popupW).
		Render(content)
}
