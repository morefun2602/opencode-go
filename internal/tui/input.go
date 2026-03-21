package tui

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type inputModel struct {
	area textarea.Model
}

func newInput(theme Theme) inputModel {
	ta := textarea.New()
	ta.Placeholder = "Type a message..."
	ta.SetHeight(2)
	ta.Focus()
	ta.CharLimit = 0
	return inputModel{area: ta}
}

func (i inputModel) Value() string {
	return i.area.Value()
}

func (i *inputModel) Reset() {
	i.area.Reset()
}

func (i inputModel) Update(msg tea.Msg) (inputModel, tea.Cmd) {
	var cmd tea.Cmd
	i.area, cmd = i.area.Update(msg)
	return i, cmd
}

func (i inputModel) View(w int, theme Theme) string {
	return lipgloss.NewStyle().
		Width(w).
		BorderTop(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(theme.Border).
		Render(i.area.View())
}
