package tui

import "github.com/charmbracelet/lipgloss"

// Theme holds the color palette for the TUI.
type Theme struct {
	Name       string
	Primary    lipgloss.Color
	Secondary  lipgloss.Color
	Background lipgloss.Color
	Text       lipgloss.Color
	Subtle     lipgloss.Color
	Error      lipgloss.Color
	Success    lipgloss.Color
	Border     lipgloss.Color
}

var Dark = Theme{
	Name:       "dark",
	Primary:    lipgloss.Color("#7C3AED"),
	Secondary:  lipgloss.Color("#3B82F6"),
	Background: lipgloss.Color("#1E1E2E"),
	Text:       lipgloss.Color("#CDD6F4"),
	Subtle:     lipgloss.Color("#6C7086"),
	Error:      lipgloss.Color("#F38BA8"),
	Success:    lipgloss.Color("#A6E3A1"),
	Border:     lipgloss.Color("#45475A"),
}

var Light = Theme{
	Name:       "light",
	Primary:    lipgloss.Color("#7C3AED"),
	Secondary:  lipgloss.Color("#3B82F6"),
	Background: lipgloss.Color("#EFF1F5"),
	Text:       lipgloss.Color("#4C4F69"),
	Subtle:     lipgloss.Color("#9CA0B0"),
	Error:      lipgloss.Color("#D20F39"),
	Success:    lipgloss.Color("#40A02B"),
	Border:     lipgloss.Color("#BCC0CC"),
}
