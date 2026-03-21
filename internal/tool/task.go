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

func registerTask(reg *tools.Registry, runner TaskRunner, workspaceID string, maxDepth int) {
	if maxDepth <= 0 {
		maxDepth = 2
	}
	type depthKey struct{}
	reg.Register(tools.Tool{
		Name:        "task",
		Description: "Run a sub-agent to complete a task and return its output",
		Tags:        []string{"execute"},
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"prompt": map[string]any{"type": "string", "description": "task description for the sub-agent"},
			},
			"required": []string{"prompt"},
		},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			depth, _ := ctx.Value(depthKey{}).(int)
			if depth >= maxDepth {
				return "", fmt.Errorf("exceeded max task nesting depth (%d)", maxDepth)
			}
			prompt := fmt.Sprint(args["prompt"])
			sid, err := runner.CreateSession(ctx, workspaceID)
			if err != nil {
				return "", err
			}
			sub := context.WithValue(ctx, depthKey{}, depth+1)
			return runner.CompleteTurn(sub, workspaceID, sid, prompt)
		},
	})
}
