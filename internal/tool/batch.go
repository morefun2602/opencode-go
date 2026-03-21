package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/morefun2602/opencode-go/internal/tools"
)

type batchCall struct {
	Tool string         `json:"tool"`
	Args map[string]any `json:"args"`
}

type batchResult struct {
	Tool   string `json:"tool"`
	Index  int    `json:"index"`
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}

func registerBatch(reg *tools.Registry) {
	reg.Register(tools.Tool{
		Name:        "batch",
		Description: "Execute multiple tool calls in a single invocation. Read tools run concurrently; write/execute tools run sequentially.",
		Tags:        []string{"execute"},
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"calls": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"tool": map[string]any{"type": "string"},
							"args": map[string]any{"type": "object"},
						},
						"required": []string{"tool", "args"},
					},
					"description": "array of {tool, args} to execute (max 25)",
				},
			},
			"required": []string{"calls"},
		},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			callsRaw, ok := args["calls"]
			if !ok {
				return "", fmt.Errorf("missing 'calls' parameter")
			}
			b, err := json.Marshal(callsRaw)
			if err != nil {
				return "", fmt.Errorf("invalid calls format: %w", err)
			}
			var calls []batchCall
			if err := json.Unmarshal(b, &calls); err != nil {
				return "", fmt.Errorf("invalid calls format: %w", err)
			}
			if len(calls) == 0 {
				return "", fmt.Errorf("calls array is empty")
			}
			if len(calls) > 25 {
				return "", fmt.Errorf("batch supports at most 25 calls, got %d", len(calls))
			}

			var readCalls []int
			var writeCalls []int
			for i, c := range calls {
				t, ok := reg.Get(c.Tool)
				if !ok {
					return "", fmt.Errorf("call[%d]: unknown tool %q", i, c.Tool)
				}
				if isReadOnly(t.Tags) {
					readCalls = append(readCalls, i)
				} else {
					writeCalls = append(writeCalls, i)
				}
			}

			results := make([]batchResult, len(calls))

			sessionID, _ := ctx.Value(tools.SessionKey).(string)

			var wg sync.WaitGroup
			for _, idx := range readCalls {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					out, err := reg.Run(ctx, "", sessionID, calls[i].Tool, calls[i].Args)
					results[i] = batchResult{Tool: calls[i].Tool, Index: i, Result: out}
					if err != nil {
						results[i].Error = err.Error()
					}
				}(idx)
			}
			wg.Wait()

			for _, idx := range writeCalls {
				out, err := reg.Run(ctx, "", sessionID, calls[idx].Tool, calls[idx].Args)
				results[idx] = batchResult{Tool: calls[idx].Tool, Index: idx, Result: out}
				if err != nil {
					results[idx].Error = err.Error()
				}
			}

			var sb strings.Builder
			for _, r := range results {
				if r.Error != "" {
					fmt.Fprintf(&sb, "[%d] %s: ERROR: %s\n", r.Index, r.Tool, r.Error)
				} else {
					fmt.Fprintf(&sb, "[%d] %s: %s\n", r.Index, r.Tool, r.Result)
				}
			}
			return sb.String(), nil
		},
	})
}

func isReadOnly(tags []string) bool {
	for _, t := range tags {
		if t == "write" || t == "execute" {
			return false
		}
	}
	return true
}
