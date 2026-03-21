package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/morefun2602/opencode-go/internal/tools"
)

type editPair struct {
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

func registerMultiedit(reg *tools.Registry, root string) {
	reg.Register(tools.Tool{
		Name:        "multiedit",
		Description: "Apply multiple find-and-replace edits to a single file atomically",
		Tags:        []string{"write"},
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string", "description": "file path"},
				"edits": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"old_string": map[string]any{"type": "string"},
							"new_string": map[string]any{"type": "string"},
						},
						"required": []string{"old_string", "new_string"},
					},
					"description": "array of {old_string, new_string} pairs",
				},
			},
			"required": []string{"path", "edits"},
		},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			p := fmt.Sprint(args["path"])
			rp, err := ResolveUnder(root, p)
			if err != nil {
				return "", err
			}

			editsRaw, ok := args["edits"]
			if !ok {
				return "", fmt.Errorf("missing 'edits' parameter")
			}
			b, err := json.Marshal(editsRaw)
			if err != nil {
				return "", fmt.Errorf("invalid edits format: %w", err)
			}
			var edits []editPair
			if err := json.Unmarshal(b, &edits); err != nil {
				return "", fmt.Errorf("invalid edits format: %w", err)
			}
			if len(edits) == 0 {
				return "", fmt.Errorf("edits array is empty")
			}

			content, err := os.ReadFile(rp)
			if err != nil {
				return "", err
			}
			text := string(content)

			for i, e := range edits {
				n := strings.Count(text, e.OldString)
				if n == 0 {
					return "", fmt.Errorf("edit[%d]: old_string not found in %s", i, p)
				}
				if n > 1 {
					return "", fmt.Errorf("edit[%d]: old_string appears %d times in %s (must be unique)", i, n, p)
				}
			}

			for _, e := range edits {
				text = strings.Replace(text, e.OldString, e.NewString, 1)
			}

			if err := os.WriteFile(rp, []byte(text), 0o644); err != nil {
				return "", err
			}
			return fmt.Sprintf("ok: %d edits applied", len(edits)), nil
		},
	})
}
