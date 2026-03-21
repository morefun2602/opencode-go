package mcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	mcpproto "github.com/mark3labs/mcp-go/mcp"
)

var (
	ErrNotConnected = errors.New("mcp: not connected")
	ErrUnknownTool  = errors.New("mcp: unknown tool")
)

type Tool struct {
	Name        string
	Description string
	Schema      map[string]any
}

type Client struct {
	inner    *mcpclient.Client
	ServerID string
	Prefix   string
	Log      *slog.Logger
	tools    []Tool
}

func NewClient(inner *mcpclient.Client, id, prefix string, log *slog.Logger) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := inner.Initialize(ctx, mcpproto.InitializeRequest{
		Params: mcpproto.InitializeParams{
			ProtocolVersion: mcpproto.LATEST_PROTOCOL_VERSION,
			ClientInfo:      mcpproto.Implementation{Name: "opencode-go", Version: "0.1.0"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("mcp initialize %s: %w", id, err)
	}

	result, err := inner.ListTools(ctx, mcpproto.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("mcp list tools %s: %w", id, err)
	}

	tools := make([]Tool, 0, len(result.Tools))
	for _, t := range result.Tools {
		schema := map[string]any{
			"type": t.InputSchema.Type,
		}
		if t.InputSchema.Properties != nil {
			schema["properties"] = t.InputSchema.Properties
		}
		if t.InputSchema.Required != nil {
			schema["required"] = t.InputSchema.Required
		}
		tools = append(tools, Tool{Name: t.Name, Description: t.Description, Schema: schema})
	}

	return &Client{inner: inner, ServerID: id, Prefix: prefix, Log: log, tools: tools}, nil
}

func (c *Client) ListTools() []Tool {
	p := c.Prefix
	if p == "" {
		p = "mcp."
	}
	sid := c.ServerID
	if sid == "" {
		sid = "default"
	}
	out := make([]Tool, 0, len(c.tools))
	for _, t := range c.tools {
		full := p + sid + "." + t.Name
		out = append(out, Tool{Name: full, Description: t.Description, Schema: t.Schema})
	}
	return out
}

func (c *Client) CallTool(ctx context.Context, fullName string, args map[string]any) (string, error) {
	p := c.Prefix
	if p == "" {
		p = "mcp."
	}
	sid := c.ServerID
	if sid == "" {
		sid = "default"
	}
	prefix := p + sid + "."
	if !strings.HasPrefix(fullName, prefix) {
		return "", ErrUnknownTool
	}
	base := strings.TrimPrefix(fullName, prefix)

	req := mcpproto.CallToolRequest{}
	req.Params.Name = base
	req.Params.Arguments = args

	var last error
	for attempt := range 3 {
		result, err := c.inner.CallTool(ctx, req)
		if err == nil {
			var b strings.Builder
			for _, content := range result.Content {
				if tc, ok := content.(mcpproto.TextContent); ok {
					b.WriteString(tc.Text)
				}
			}
			return b.String(), nil
		}
		last = err
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if !isRetryable(err) {
			break
		}
		if c.Log != nil {
			c.Log.Warn("mcp_retry", "attempt", attempt+1, "err", err)
		}
		time.Sleep(time.Duration(50*(attempt+1)) * time.Millisecond)
	}
	return "", last
}

func (c *Client) Close() error {
	if c.inner != nil {
		return c.inner.Close()
	}
	return nil
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "timeout") || strings.Contains(s, "Temporary")
}
