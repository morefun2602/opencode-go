package tool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/morefun2602/opencode-go/internal/tools"
)

func registerTodowrite(reg *tools.Registry) {
	reg.Register(tools.Tool{
		Name:        "todowrite",
		Description: "Create or update a structured todo list for tracking tasks",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"todos": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"id":      map[string]any{"type": "string"},
							"content": map[string]any{"type": "string"},
							"status":  map[string]any{"type": "string", "enum": []string{"pending", "in_progress", "completed"}},
						},
						"required": []string{"id", "content", "status"},
					},
				},
				"merge": map[string]any{"type": "boolean"},
			},
			"required": []string{"todos"},
		},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			session, _ := ctx.Value(tools.SessionKey).(string)
			if session == "" {
				session = "default"
			}
			raw, err := json.Marshal(args["todos"])
			if err != nil {
				return "", fmt.Errorf("todowrite: invalid todos: %w", err)
			}
			var items []tools.TodoItem
			if err := json.Unmarshal(raw, &items); err != nil {
				return "", fmt.Errorf("todowrite: %w", err)
			}
			if argBool(args, "merge") {
				tools.GlobalTodos.Merge(session, items)
			} else {
				tools.GlobalTodos.Set(session, items)
			}
			return tools.FormatTodos(session), nil
		},
	})
}
