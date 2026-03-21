package tool

import (
	"context"
	"log/slog"
	"strings"

	"github.com/morefun2602/opencode-go/internal/mcp"
	"github.com/morefun2602/opencode-go/internal/tools"
)

// Router 统一解析内置与 MCP 工具名；内置同名优先于 MCP。
type Router struct {
	Builtin *tools.Registry
	Clients []*mcp.Client
	Log     *slog.Logger
}

// Run 执行工具；对未找到的工具名先尝试小写匹配，最后路由到 invalid 工具。
func (r *Router) Run(ctx context.Context, corrID, sessionID, name string, args map[string]any) (string, error) {
	if r.Builtin != nil && r.Builtin.Has(name) {
		return r.Builtin.Run(ctx, corrID, sessionID, name, args)
	}
	for _, c := range r.Clients {
		for _, t := range c.ListTools() {
			if t.Name == name {
				out, err := c.CallTool(ctx, name, args)
				if err != nil && r.Log != nil {
					r.Log.Error("mcp_tool_fail", "tool", name, "corr_id", corrID, "session_id", sessionID, "err", err)
				}
				return out, err
			}
		}
	}

	lower := strings.ToLower(name)
	if lower != name {
		if r.Builtin != nil && r.Builtin.Has(lower) {
			if r.Log != nil {
				r.Log.Info("tool_name_repaired", "original", name, "repaired", lower)
			}
			return r.Builtin.Run(ctx, corrID, sessionID, lower, args)
		}
		for _, c := range r.Clients {
			for _, t := range c.ListTools() {
				if strings.EqualFold(t.Name, name) {
					if r.Log != nil {
						r.Log.Info("tool_name_repaired", "original", name, "repaired", t.Name)
					}
					out, err := c.CallTool(ctx, t.Name, args)
					if err != nil && r.Log != nil {
						r.Log.Error("mcp_tool_fail", "tool", t.Name, "corr_id", corrID, "session_id", sessionID, "err", err)
					}
					return out, err
				}
			}
		}
	}

	if r.Builtin != nil && r.Builtin.Has("invalid") {
		return r.Builtin.Run(ctx, corrID, sessionID, "invalid", map[string]any{
			"tool":  name,
			"error": "unknown tool",
		})
	}
	return "", &ErrUnknown{Name: name}
}
