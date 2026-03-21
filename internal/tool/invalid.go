package tool

import (
	"context"
	"fmt"

	"github.com/morefun2602/opencode-go/internal/tools"
)

const invalidToolName = "invalid"

func registerInvalid(reg *tools.Registry) {
	reg.Register(tools.Tool{
		Name:        invalidToolName,
		Description: "Handles malformed or invalid tool calls",
		Tags:        []string{"internal"},
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"tool":  map[string]any{"type": "string", "description": "original tool name"},
				"error": map[string]any{"type": "string", "description": "error description"},
			},
			"required": []string{"tool", "error"},
		},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			toolName := fmt.Sprint(args["tool"])
			errMsg := fmt.Sprint(args["error"])
			return fmt.Sprintf("The arguments provided to tool '%s' are invalid: %s", toolName, errMsg), nil
		},
	})
}
