package tool

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/morefun2602/opencode-go/internal/tools"
)

func registerWebfetch(reg *tools.Registry, timeout int) {
	if timeout <= 0 {
		timeout = 30
	}
	reg.Register(tools.Tool{
		Name:        "webfetch",
		Description: "Fetch a URL and return its text content",
		Tags:        []string{"read"},
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]any{"type": "string", "description": "URL to fetch"},
			},
			"required": []string{"url"},
		},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			u := fmt.Sprint(args["url"])
			cctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
			defer cancel()
			req, err := http.NewRequestWithContext(cctx, "GET", u, nil)
			if err != nil {
				return "", err
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return "", err
			}
			defer resp.Body.Close()
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return "", fmt.Errorf("webfetch: HTTP %d", resp.StatusCode)
			}
			b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			if err != nil {
				return "", err
			}
			s := stripHTML(string(b))
			return s, nil
		},
	})
}

func stripHTML(s string) string {
	if !strings.Contains(s, "<") {
		return s
	}
	var out strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			out.WriteRune(r)
		}
	}
	return out.String()
}
