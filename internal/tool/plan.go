package tool

import (
	"context"
	"fmt"

	"github.com/morefun2602/opencode-go/internal/tools"
)

// PlanSwitch abstracts session-level plan mode switching to avoid importing runtime.
type PlanSwitch interface {
	IsInPlan(sessionID string) bool
	EnterPlan(sessionID string)
	ExitPlan(sessionID string)
}

// RegisterPlan registers plan_enter and plan_exit tools that operate on the
// given PlanSwitch to toggle between build and plan modes per session.
func RegisterPlan(reg *tools.Registry, ps PlanSwitch) {
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
			if ps.IsInPlan(sessionID) {
				return "already in plan mode", nil
			}
			ps.EnterPlan(sessionID)
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
			if !ps.IsInPlan(sessionID) {
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

			ps.ExitPlan(sessionID)
			return "switched to build mode", nil
		},
	})
}
