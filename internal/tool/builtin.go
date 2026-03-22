package tool

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/morefun2602/opencode-go/internal/filewatcher"
	"github.com/morefun2602/opencode-go/internal/policy"
	"github.com/morefun2602/opencode-go/internal/skill"
	"github.com/morefun2602/opencode-go/internal/tools"
)

// RegisterBuiltin 注册内置工具集。
// task 工具需要 Engine 引用，通过 RegisterTask 单独注册。
func RegisterBuiltin(reg *tools.Registry, pol *policy.Policy, skills []skill.Skill, watcher *filewatcher.Watcher) {
	root := "."
	if pol != nil && pol.WorkspaceRoot != "" {
		root = pol.WorkspaceRoot
	}
	maxOut := 256 * 1024
	if pol != nil && pol.MaxOutputBytes > 0 {
		maxOut = pol.MaxOutputBytes
	}
	bashSec := 30
	if pol != nil && pol.BashTimeoutSec > 0 {
		bashSec = pol.BashTimeoutSec
	}
	reg.Register(tools.Tool{
		Name:        "read",
		Description: "Read file contents with optional offset and limit",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":   map[string]any{"type": "string", "description": "file path"},
				"offset": map[string]any{"type": "integer", "description": "start line (1-based, optional)"},
				"limit":  map[string]any{"type": "integer", "description": "number of lines to read (optional)"},
			},
			"required": []string{"path"},
		},
		Tags: []string{"read"},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			p := fmt.Sprint(args["path"])
			rp, err := ResolveUnder(root, p)
			if err != nil {
				return "", err
			}
			b, err := os.ReadFile(rp)
			if err != nil {
				return "", err
			}
			lines := strings.Split(string(b), "\n")

			offset := 0
			if v, ok := args["offset"]; ok {
				if f, ok := v.(float64); ok && f > 0 {
					offset = int(f) - 1
				}
			}
			limit := len(lines)
			if v, ok := args["limit"]; ok {
				if f, ok := v.(float64); ok && f > 0 {
					limit = int(f)
				}
			}

			if offset >= len(lines) {
				return fmt.Sprintf("(file has %d lines, offset %d is beyond end)", len(lines), offset+1), nil
			}
			end := offset + limit
			if end > len(lines) {
				end = len(lines)
			}
			selected := lines[offset:end]

			const maxLineLen = 2000
			for i, line := range selected {
				if len(line) > maxLineLen {
					selected[i] = line[:maxLineLen] + "... (line truncated)"
				}
				selected[i] = fmt.Sprintf("%6d|%s", offset+i+1, selected[i])
			}
			return strings.Join(selected, "\n"), nil
		},
	})
	reg.Register(tools.Tool{
		Name:   "write",
		Schema: map[string]any{"path": "string", "content": "string", "confirm": "bool"},
		Tags:   []string{"write"},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			if pol != nil && pol.RequireWriteConfirm {
				if !argBool(args, "confirm") {
					return "", fmt.Errorf("write blocked: confirmation required")
				}
			}
			p := fmt.Sprint(args["path"])
			rp, err := ResolveUnder(root, p)
			if err != nil {
				return "", err
			}
			if err := os.MkdirAll(filepath.Dir(rp), 0o755); err != nil {
				return "", err
			}
			content := fmt.Sprint(args["content"])
			if err := os.WriteFile(rp, []byte(content), 0o644); err != nil {
				return "", err
			}
			if pol != nil {
				pol.Audit("tool_write", "path", rp)
			}
			if watcher != nil {
				watcher.NotifyChange(rp)
			}
			return "ok", nil
		},
	})
	reg.Register(tools.Tool{
		Name:   "glob",
		Schema: map[string]any{"pattern": "string"},
		Tags:   []string{"read"},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			pat := fmt.Sprint(args["pattern"])
			base, err := filepath.Abs(root)
			if err != nil {
				return "", err
			}
			matches, err := filepath.Glob(filepath.Join(base, pat))
			if err != nil {
				return "", err
			}
			return strings.Join(matches, "\n"), nil
		},
	})
	reg.Register(tools.Tool{
		Name:   "grep",
		Schema: map[string]any{"path": "string", "pattern": "string"},
		Tags:   []string{"read"},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			p := fmt.Sprint(args["path"])
			rp, err := ResolveUnder(root, p)
			if err != nil {
				return "", err
			}
			b, err := os.ReadFile(rp)
			if err != nil {
				return "", err
			}
			line := fmt.Sprint(args["pattern"])
			var hits []string
			for i, s := range strings.Split(string(b), "\n") {
				if strings.Contains(s, line) {
					hits = append(hits, fmt.Sprintf("%d:%s", i+1, s))
				}
			}
			return strings.Join(hits, "\n"), nil
		},
	})
	reg.Register(tools.Tool{
		Name:   "bash",
		Schema: map[string]any{"cmd": "string"},
		Tags:   []string{"execute"},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			cmdline := fmt.Sprint(args["cmd"])
			cctx, cancel := context.WithTimeout(ctx, time.Duration(bashSec)*time.Second)
			defer cancel()
			cmd := exec.CommandContext(cctx, "sh", "-c", cmdline)
			cmd.Dir = root
			out, err := cmd.CombinedOutput()
			s := string(out)
			if err != nil {
				return s, fmt.Errorf("bash: %w", err)
			}
			if pol != nil {
				pol.Audit("tool_bash", "cmd", cmdline)
			}
			return s, nil
		},
	})
	registerEdit(reg, root, watcher)
	registerWebfetch(reg, bashSec)
	registerTodowrite(reg)
	registerTodoread(reg)
	registerApplyPatch(reg, root, watcher)
	registerQuestion(reg)
	searchURL := ""
	if pol != nil {
		searchURL = pol.SearchURL
	}
	registerWebsearch(reg, searchURL, bashSec, maxOut)
	registerMultiedit(reg, root)
	registerBatch(reg)
	registerSkillTool(reg, skills)
	registerLs(reg, root)
	registerInvalid(reg)
}

func argBool(m map[string]any, k string) bool {
	v, ok := m[k]
	if !ok {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case string:
		return t == "true" || t == "1"
	default:
		return false
	}
}
