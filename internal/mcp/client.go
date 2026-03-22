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

type Resource struct {
	Name        string
	URI         string
	Description string
	MimeType    string
}

type Client struct {
	inner         *mcpclient.Client
	ServerID      string
	Prefix        string
	Log           *slog.Logger
	tools         []Tool
	resources     []Resource
	oauthProvider *OAuthProvider
	timeout       time.Duration
}

func NewClient(inner *mcpclient.Client, id, prefix string, log *slog.Logger, oauthProvider *OAuthProvider, timeout time.Duration) (*Client, error) {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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

	resourcesResult, err := inner.ListResources(ctx, mcpproto.ListResourcesRequest{})
	var resources []Resource
	if err != nil {
		// Some MCP servers do not implement resources. Keep tools available.
		if log != nil {
			log.Warn("mcp_list_resources_failed", "server", id, "err", err)
		}
	} else {
		resources = make([]Resource, 0, len(resourcesResult.Resources))
		for _, r := range resourcesResult.Resources {
			resources = append(resources, Resource{
				Name:        r.Name,
				URI:         r.URI,
				Description: r.Description,
				MimeType:    r.MIMEType,
			})
		}
	}

	return &Client{
		inner:         inner,
		ServerID:      id,
		Prefix:        prefix,
		Log:           log,
		tools:         tools,
		resources:     resources,
		oauthProvider: oauthProvider,
		timeout:       timeout,
	}, nil
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
		if c.isUnauthorized(err) && c.oauthProvider != nil {
			if c.Log != nil {
				c.Log.Warn("mcp_oauth_retry", "server", c.ServerID, "tool", fullName, "attempt", attempt+1)
			}
			_ = c.oauthProvider.InvalidateToken()
			if _, tokenErr := c.oauthProvider.GetToken(ctx); tokenErr != nil {
				return "", fmt.Errorf("oauth refresh failed: %w", tokenErr)
			}
			continue
		}
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

func (c *Client) ListResources() []Resource {
	out := make([]Resource, len(c.resources))
	copy(out, c.resources)
	return out
}

func (c *Client) ReadResource(ctx context.Context, uri string, arguments map[string]any) (string, error) {
	req := mcpproto.ReadResourceRequest{}
	req.Params.URI = uri
	if len(arguments) > 0 {
		req.Params.Arguments = arguments
	}
	var last error
	for attempt := range 3 {
		result, err := c.inner.ReadResource(ctx, req)
		if err == nil {
			var b strings.Builder
			for _, content := range result.Contents {
				switch v := any(content).(type) {
				case mcpproto.TextResourceContents:
					b.WriteString(v.Text)
				default:
					raw := fmt.Sprint(content)
					if raw != "" {
						b.WriteString(raw)
					}
				}
			}
			return b.String(), nil
		}
		last = err
		if c.isUnauthorized(err) && c.oauthProvider != nil {
			_ = c.oauthProvider.InvalidateToken()
			if _, tokenErr := c.oauthProvider.GetToken(ctx); tokenErr != nil {
				return "", fmt.Errorf("oauth refresh failed: %w", tokenErr)
			}
			continue
		}
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if !isRetryable(err) {
			break
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

func (c *Client) isUnauthorized(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "401") || strings.Contains(s, "unauthorized")
}
