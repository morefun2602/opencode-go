package tool

import (
	"context"

	"github.com/morefun2602/opencode-go/internal/tools"
)

func registerTodoread(reg *tools.Registry) {
	reg.Register(tools.Tool{
		Name:        "todoread",
		Description: "Read the current structured todo list for this session",
		Schema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		Tags: []string{"read"},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			session, _ := ctx.Value(tools.SessionKey).(string)
			if session == "" {
				session = "default"
			}
			return tools.FormatTodos(session), nil
		},
	})
}
