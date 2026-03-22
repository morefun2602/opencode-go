package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ToastLevel indicates the severity of a toast notification.
type ToastLevel int

const (
	ToastInfo ToastLevel = iota
	ToastSuccess
	ToastWarning
	ToastError
)

// Toast represents a single notification message.
type Toast struct {
	Message string
	Level   ToastLevel
	Expires time.Time
}

type toastExpired struct{ id int }

// ToastManager manages transient notification messages.
type ToastManager struct {
	toasts []Toast
	nextID int
}

func newToastManager() ToastManager {
	return ToastManager{}
}

// Add adds a toast and returns a tea.Cmd that will expire it after the given duration.
func (tm *ToastManager) Add(msg string, level ToastLevel, duration time.Duration) tea.Cmd {
	t := Toast{
		Message: msg,
		Level:   level,
		Expires: time.Now().Add(duration),
	}
	tm.toasts = append(tm.toasts, t)
	id := tm.nextID
	tm.nextID++
	return tea.Tick(duration, func(time.Time) tea.Msg {
		return toastExpired{id: id}
	})
}

// Expire removes the oldest toast.
func (tm *ToastManager) Expire() {
	if len(tm.toasts) > 0 {
		tm.toasts = tm.toasts[1:]
	}
}

// HasToasts returns true if there are active toasts.
func (tm *ToastManager) HasToasts() bool {
	return len(tm.toasts) > 0
}

// View renders all active toasts stacked at the top-right.
func (tm *ToastManager) View(w int, theme Theme) string {
	if len(tm.toasts) == 0 {
		return ""
	}

	toastW := min(50, w-4)
	var rendered []string
	for _, t := range tm.toasts {
		var icon string
		var color lipgloss.Color
		switch t.Level {
		case ToastSuccess:
			icon = "✓"
			color = theme.Success
		case ToastWarning:
			icon = "⚠"
			color = theme.Warning
		case ToastError:
			icon = "✗"
			color = theme.Error
		default:
			icon = "ℹ"
			color = theme.Info
		}

		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(color).
			Foreground(theme.Text).
			Background(theme.BackgroundPanel).
			Padding(0, 1).
			Width(toastW).
			Render(lipgloss.NewStyle().Foreground(color).Render(icon) + " " + t.Message)

		rendered = append(rendered, box)
	}

	var result string
	for _, r := range rendered {
		result += r + "\n"
	}
	return result
}
