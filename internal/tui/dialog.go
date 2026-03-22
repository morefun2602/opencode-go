package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Dialog is the interface for modal dialogs.
type Dialog interface {
	Update(msg tea.Msg) (Dialog, tea.Cmd)
	View(w, h int, theme Theme) string
	Done() bool
	Result() any
}

// DialogStack manages a stack of modal dialogs.
type DialogStack struct {
	stack []Dialog
}

func (ds *DialogStack) Push(d Dialog)  { ds.stack = append(ds.stack, d) }
func (ds *DialogStack) Top() Dialog    { return ds.stack[len(ds.stack)-1] }
func (ds *DialogStack) Empty() bool    { return len(ds.stack) == 0 }

func (ds *DialogStack) Pop() Dialog {
	n := len(ds.stack)
	d := ds.stack[n-1]
	ds.stack = ds.stack[:n-1]
	return d
}

// --- Confirm Dialog ---

type ConfirmDialog struct {
	title string
	desc  string
	done  bool
	yes   bool
}

func NewConfirmDialog(title, desc string) *ConfirmDialog {
	return &ConfirmDialog{title: title, desc: desc}
}

func (d *ConfirmDialog) Update(msg tea.Msg) (Dialog, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "y", "Y":
			d.done = true
			d.yes = true
		case "n", "N", "esc":
			d.done = true
			d.yes = false
		}
	}
	return d, nil
}

func (d *ConfirmDialog) View(w, h int, theme Theme) string {
	boxW := min(50, w-4)
	title := lipgloss.NewStyle().Bold(true).Foreground(theme.Primary).Render(d.title)
	desc := lipgloss.NewStyle().Foreground(theme.Text).Width(boxW - 4).Render(d.desc)
	hint := lipgloss.NewStyle().Foreground(theme.Subtle).Render("[y] yes  [n] no  [esc] cancel")

	content := title + "\n\n" + desc + "\n\n" + hint
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Primary).
		Padding(1, 2).
		Width(boxW).
		Render(content)

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(theme.DialogOverlay))
}

func (d *ConfirmDialog) Done() bool { return d.done }
func (d *ConfirmDialog) Result() any { return d.yes }

// --- Select Dialog ---

type dialogKind int

const (
	dialogKindGeneric dialogKind = iota
	dialogKindAgent
	dialogKindSession
	dialogKindTheme
	dialogKindModel
)

type SelectDialog struct {
	title    string
	items    []string
	kind     dialogKind
	cursor   int
	done     bool
	selected string
}

func NewSelectDialog(title string, items []string) *SelectDialog {
	return &SelectDialog{title: title, items: items, kind: dialogKindGeneric}
}

func NewSelectDialogWithKind(title string, items []string, kind dialogKind) *SelectDialog {
	return &SelectDialog{title: title, items: items, kind: kind}
}

func (d *SelectDialog) Update(msg tea.Msg) (Dialog, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "up", "k":
			if d.cursor > 0 {
				d.cursor--
			}
		case "down", "j":
			if d.cursor < len(d.items)-1 {
				d.cursor++
			}
		case "enter":
			if d.cursor < len(d.items) {
				d.done = true
				d.selected = d.items[d.cursor]
			}
		case "esc":
			d.done = true
			d.selected = ""
		}
	}
	return d, nil
}

func (d *SelectDialog) View(w, h int, theme Theme) string {
	boxW := min(50, w-4)
	title := lipgloss.NewStyle().Bold(true).Foreground(theme.Primary).Render(d.title)

	var rows []string
	for i, item := range d.items {
		style := lipgloss.NewStyle().Padding(0, 1).Width(boxW - 4)
		if i == d.cursor {
			style = style.Foreground(theme.Text).Background(theme.Border)
		} else {
			style = style.Foreground(theme.Subtle)
		}
		rows = append(rows, style.Render(item))
	}

	hint := lipgloss.NewStyle().Foreground(theme.Subtle).Render("[j/k] navigate  [enter] select  [esc] cancel")
	content := title + "\n\n" + strings.Join(rows, "\n") + "\n\n" + hint

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Primary).
		Padding(1, 2).
		Width(boxW).
		Render(content)

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(theme.DialogOverlay))
}

func (d *SelectDialog) Done() bool  { return d.done }
func (d *SelectDialog) Result() any { return d.selected }

// --- Alert Dialog ---

type AlertDialog struct {
	title string
	body  string
	done  bool
}

func NewAlertDialog(title, body string) *AlertDialog {
	return &AlertDialog{title: title, body: body}
}

func (d *AlertDialog) Update(msg tea.Msg) (Dialog, tea.Cmd) {
	if _, ok := msg.(tea.KeyMsg); ok {
		d.done = true
	}
	return d, nil
}

func (d *AlertDialog) View(w, h int, theme Theme) string {
	boxW := min(50, w-4)
	title := lipgloss.NewStyle().Bold(true).Foreground(theme.Primary).Render(d.title)
	body := lipgloss.NewStyle().Foreground(theme.Text).Width(boxW - 4).Render(d.body)
	hint := lipgloss.NewStyle().Foreground(theme.Subtle).Render("press any key to close")

	content := title + "\n\n" + body + "\n\n" + hint
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Primary).
		Padding(1, 2).
		Width(boxW).
		Render(content)

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(theme.DialogOverlay))
}

func (d *AlertDialog) Done() bool  { return d.done }
func (d *AlertDialog) Result() any { return nil }
