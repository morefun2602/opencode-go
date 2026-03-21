package tool

import (
	"context"
	"log/slog"

	"github.com/morefun2602/opencode-go/internal/mcp"
	"github.com/morefun2602/opencode-go/internal/tools"
)

// Router 统一解析内置与 MCP 工具名；内置同名优先于 MCP。
type Router struct {
	Builtin *tools.Registry
	Clients []*mcp.Client
	Log     *slog.Logger
}

// Run 执行工具；未知名返回 *ErrUnknown。
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
	return "", &ErrUnknown{Name: name}
}
