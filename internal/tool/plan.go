package tool

import (
	"context"
	"fmt"
	"sync"

	"github.com/morefun2602/opencode-go/internal/tools"
)

// ModeSwitch tracks per-session agent mode for runtime switching.
type ModeSwitch struct {
	mu    sync.RWMutex
	modes map[string]string // sessionID -> mode name
}

var GlobalModeSwitch = &ModeSwitch{modes: map[string]string{}}

func (ms *ModeSwitch) Get(sessionID string) string {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return ms.modes[sessionID]
}

func (ms *ModeSwitch) Set(sessionID, mode string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.modes[sessionID] = mode
}

func registerPlan(reg *tools.Registry) {
	reg.Register(tools.Tool{
		Name:        "plan_enter",
		Description: "Switch the current session to plan mode (read-only, no code changes)",
		Tags:        []string{"interact"},
		Schema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			sessionID, _ := ctx.Value(tools.SessionKey).(string)
			if sessionID == "" {
				return "", fmt.Errorf("no session context")
			}
			current := GlobalModeSwitch.Get(sessionID)
			if current == "plan" {
				return "already in plan mode", nil
			}
			GlobalModeSwitch.Set(sessionID, "plan")
			return "switched to plan mode", nil
		},
	})

	reg.Register(tools.Tool{
		Name:        "plan_exit",
		Description: "Exit plan mode and switch back to build mode (requires user confirmation)",
		Tags:        []string{"interact"},
		Schema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			sessionID, _ := ctx.Value(tools.SessionKey).(string)
			if sessionID == "" {
				return "", fmt.Errorf("no session context")
			}
			current := GlobalModeSwitch.Get(sessionID)
			if current != "plan" {
				return "not in plan mode", nil
			}

			answer, err := tools.Questions.Ask(ctx, "plan_exit_confirm",
				"Are you sure you want to exit plan mode and switch to build mode?",
				[]string{"yes", "no"})
			if err != nil {
				return "", fmt.Errorf("confirmation failed: %w", err)
			}
			if answer != "yes" {
				return "staying in plan mode", nil
			}

			GlobalModeSwitch.Set(sessionID, "build")
			return "switched to build mode", nil
		},
	})
}
