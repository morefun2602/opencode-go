package tool

import (
	"context"
	"fmt"

	"github.com/morefun2602/opencode-go/internal/filewatcher"
	"github.com/morefun2602/opencode-go/internal/tools"
)

func registerApplyPatch(reg *tools.Registry, root string, watcher *filewatcher.Watcher) {
	reg.Register(tools.Tool{
		Name:        "apply_patch",
		Description: "Apply a unified diff patch to files in the workspace",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"patch": map[string]any{"type": "string", "description": "unified diff content"},
			},
			"required": []string{"patch"},
		},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			patch := fmt.Sprint(args["patch"])
			fps, err := tools.ParsePatch(patch)
			if err != nil {
				return "", fmt.Errorf("apply_patch: parse: %w", err)
			}
			if len(fps) == 0 {
				return "", fmt.Errorf("apply_patch: no file patches found")
			}
			resolve := func(p string) (string, error) {
				return ResolveUnder(root, p)
			}
			if err := tools.ApplyFilePatches(fps, resolve); err != nil {
				return "", fmt.Errorf("apply_patch: %w", err)
			}
			if watcher != nil {
				for _, fp := range fps {
					p := fp.NewPath
					if p == "" {
						p = fp.OldPath
					}
					if rp, err := resolve(p); err == nil {
						watcher.NotifyChange(rp)
					}
				}
			}
			return fmt.Sprintf("applied %d file(s)", len(fps)), nil
		},
	})
}
