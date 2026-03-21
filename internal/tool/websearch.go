package tool

import (
	"context"
	"fmt"

	"github.com/morefun2602/opencode-go/internal/tools"
)

func registerWebsearch(reg *tools.Registry, searchURL string, timeout, maxOut int) {
	if timeout <= 0 {
		timeout = 30
	}
	if maxOut <= 0 {
		maxOut = 256 * 1024
	}
	reg.Register(tools.Tool{
		Name:        "websearch",
		Description: "Search the web using a configured search endpoint",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string", "description": "search query"},
			},
			"required": []string{"query"},
		},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			q := fmt.Sprint(args["query"])
			return tools.DoSearch(ctx, searchURL, q, timeout, maxOut)
		},
	})
}
