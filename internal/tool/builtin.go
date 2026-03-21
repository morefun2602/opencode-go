package tool

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/morefun2602/opencode-go/internal/policy"
	"github.com/morefun2602/opencode-go/internal/tools"
)

// RegisterBuiltin 注册内置工具集（read/write/glob/grep/bash/edit/webfetch）。
// task 工具需要 Engine 引用，通过 RegisterTask 单独注册。
func RegisterBuiltin(reg *tools.Registry, pol *policy.Policy) {
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
		Name:   "read",
		Schema: map[string]any{"path": "string"},
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
			return string(b), nil
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
			out := strings.Join(hits, "\n")
			if len(out) > maxOut {
				out = out[:maxOut] + "\n…truncated"
			}
			return out, nil
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
			if len(s) > maxOut {
				s = s[:maxOut] + "\n…truncated"
			}
			if err != nil {
				return s, fmt.Errorf("bash: %w", err)
			}
			if pol != nil {
				pol.Audit("tool_bash", "cmd", cmdline)
			}
			return s, nil
		},
	})
	registerEdit(reg, root)
	registerWebfetch(reg, bashSec, maxOut)
	registerTodowrite(reg)
	registerApplyPatch(reg, root)
	registerQuestion(reg)
	searchURL := ""
	if pol != nil {
		searchURL = pol.SearchURL
	}
	registerWebsearch(reg, searchURL, bashSec, maxOut)
}

// RegisterTask registers the task (sub-agent) tool. Called separately because it needs an Engine reference.
func RegisterTask(reg *tools.Registry, runner TaskRunner, workspaceID string, maxDepth int) {
	registerTask(reg, runner, workspaceID, maxDepth)
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
