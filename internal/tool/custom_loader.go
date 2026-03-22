package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/morefun2602/opencode-go/internal/tools"
)

type customToolFile struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Command     string         `json:"command"`
	Tags        []string       `json:"tags"`
	Schema      map[string]any `json:"schema"`
}

// RegisterCustomToolsFromWorkspace loads custom tool definitions from:
// - .opencode/tool/*.json
// - .opencode/tools/*.json
// Each tool runs `command` through `/bin/zsh -lc`, with tool args passed as JSON stdin.
func RegisterCustomToolsFromWorkspace(reg *tools.Registry, workspaceRoot string, log *slog.Logger) {
	if reg == nil || workspaceRoot == "" {
		return
	}
	patterns := []string{
		filepath.Join(workspaceRoot, ".opencode", "tool", "*.json"),
		filepath.Join(workspaceRoot, ".opencode", "tools", "*.json"),
	}
	for _, pat := range patterns {
		matches, err := filepath.Glob(pat)
		if err != nil {
			if log != nil {
				log.Warn("custom_tool_glob_failed", "pattern", pat, "err", err)
			}
			continue
		}
		for _, file := range matches {
			def, err := loadCustomToolFile(file)
			if err != nil {
				if log != nil {
					log.Warn("custom_tool_parse_failed", "file", file, "err", err)
				}
				continue
			}
			if def.Name == "" || def.Command == "" {
				if log != nil {
					log.Warn("custom_tool_invalid", "file", file, "err", "name/command required")
				}
				continue
			}
			if reg.Has(def.Name) {
				if log != nil {
					log.Warn("custom_tool_conflict", "name", def.Name, "file", file)
				}
				continue
			}
			schema := def.Schema
			if len(schema) == 0 {
				schema = map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				}
			}
			tags := def.Tags
			if len(tags) == 0 {
				tags = []string{"execute"}
			}
			toolName := def.Name
			toolCmd := def.Command
			toolDesc := def.Description
			toolTags := append([]string(nil), tags...)
			toolSchema := schema
			reg.Register(tools.Tool{
				Name:        toolName,
				Description: toolDesc,
				Schema:      toolSchema,
				Tags:        toolTags,
				Fn: func(ctx context.Context, args map[string]any) (string, error) {
					payload, _ := json.Marshal(args)
					cmd := exec.CommandContext(ctx, "/bin/zsh", "-lc", toolCmd)
					cmd.Dir = workspaceRoot
					cmd.Stdin = strings.NewReader(string(payload))
					out, err := cmd.CombinedOutput()
					if err != nil {
						return string(out), fmt.Errorf("custom tool %q failed: %w", toolName, err)
					}
					return string(out), nil
				},
			})
			if log != nil {
				log.Info("custom_tool_registered", "name", def.Name, "file", file)
			}
		}
	}
}

func loadCustomToolFile(path string) (customToolFile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return customToolFile{}, err
	}
	var def customToolFile
	if err := json.Unmarshal(b, &def); err != nil {
		return customToolFile{}, err
	}
	return def, nil
}
