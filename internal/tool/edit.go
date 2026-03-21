package tool

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/morefun2602/opencode-go/internal/filewatcher"
	"github.com/morefun2602/opencode-go/internal/tools"
)

func registerEdit(reg *tools.Registry, root string, watcher *filewatcher.Watcher) {
	reg.Register(tools.Tool{
		Name:        "edit",
		Description: "Replace a unique occurrence of old_string with new_string in a file",
		Tags:        []string{"write"},
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":       map[string]any{"type": "string", "description": "file path"},
				"old_string": map[string]any{"type": "string", "description": "text to find (must be unique)"},
				"new_string": map[string]any{"type": "string", "description": "replacement text"},
			},
			"required": []string{"path", "old_string", "new_string"},
		},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			p := fmt.Sprint(args["path"])
			rp, err := ResolveUnder(root, p)
			if err != nil {
				return "", err
			}
			old := fmt.Sprint(args["old_string"])
			neu := fmt.Sprint(args["new_string"])
			b, err := os.ReadFile(rp)
			if err != nil {
				return "", err
			}
			content := string(b)
			n := strings.Count(content, old)
			if n == 0 {
				return "", fmt.Errorf("old_string not found in %s", p)
			}
			if n > 1 {
				return "", fmt.Errorf("old_string appears %d times in %s (must be unique)", n, p)
			}
			result := strings.Replace(content, old, neu, 1)
			if err := os.WriteFile(rp, []byte(result), 0o644); err != nil {
				return "", err
			}
			if watcher != nil {
				watcher.NotifyChange(rp)
			}
			return "ok", nil
		},
	})
}
