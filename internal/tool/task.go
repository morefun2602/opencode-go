package tool

import (
	"context"
	"fmt"

	"github.com/morefun2602/opencode-go/internal/tools"
)

// TaskRunner is an interface matching Engine.CompleteTurn so we avoid import cycles.
type TaskRunner interface {
	CompleteTurn(ctx context.Context, workspaceID, sessionID, userText string) (string, error)
	CreateSession(ctx context.Context, workspaceID string) (string, error)
}

// SessionLookup checks if a session exists.
type SessionLookup interface {
	SessionExists(ctx context.Context, workspaceID, sessionID string) (bool, error)
}

func registerTask(reg *tools.Registry, runner TaskRunner, lookup SessionLookup, workspaceID string, maxDepth int) {
	if maxDepth <= 0 {
		maxDepth = 2
	}
	type depthKey struct{}
	reg.Register(tools.Tool{
		Name:        "task",
		Description: "Run a sub-agent to complete a task. Supports resuming previous tasks via task_id.",
		Tags:        []string{"execute"},
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"prompt":        map[string]any{"type": "string", "description": "task description for the sub-agent"},
				"task_id":       map[string]any{"type": "string", "description": "optional: resume an existing sub-agent session"},
				"subagent_type": map[string]any{"type": "string", "description": "optional: agent mode (build, plan, explore)"},
				"description":   map[string]any{"type": "string", "description": "optional: brief description for tracking"},
			},
			"required": []string{"prompt"},
		},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			depth, _ := ctx.Value(depthKey{}).(int)
			if depth >= maxDepth {
				return "", fmt.Errorf("exceeded max task nesting depth (%d)", maxDepth)
			}
			prompt := fmt.Sprint(args["prompt"])

			var sid string
			taskID, _ := args["task_id"].(string)

			if taskID != "" {
				exists, err := lookup.SessionExists(ctx, workspaceID, taskID)
				if err != nil {
					return "", fmt.Errorf("checking task_id: %w", err)
				}
				if !exists {
					return "", fmt.Errorf("task_id %q not found", taskID)
				}
				sid = taskID
			} else {
				var err error
				sid, err = runner.CreateSession(ctx, workspaceID)
				if err != nil {
					return "", err
				}
			}

			sub := context.WithValue(ctx, depthKey{}, depth+1)
			result, err := runner.CompleteTurn(sub, workspaceID, sid, prompt)
			if err != nil {
				return "", err
			}

			return fmt.Sprintf("task_id: %s\n%s", sid, result), nil
		},
	})
}

// RegisterTask registers the task (sub-agent) tool. Called separately because it needs an Engine reference.
func RegisterTask(reg *tools.Registry, runner TaskRunner, lookup SessionLookup, workspaceID string, maxDepth int) {
	registerTask(reg, runner, lookup, workspaceID, maxDepth)
}
