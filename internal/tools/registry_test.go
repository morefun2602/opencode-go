package tools

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
)

func TestRegistryRunValidatesRequiredField(t *testing.T) {
	reg := New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	reg.Register(Tool{
		Name: "demo",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string"},
			},
			"required": []string{"path"},
		},
		Fn: func(ctx context.Context, args map[string]any) (string, error) { return "ok", nil },
	})

	_, err := reg.Run(context.Background(), "corr", "session", "demo", map[string]any{})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "missing required field") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegistryRunValidatesEnum(t *testing.T) {
	reg := New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	reg.Register(Tool{
		Name: "todo",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"status": map[string]any{
					"type": "string",
					"enum": []string{"pending", "done"},
				},
			},
			"required": []string{"status"},
		},
		Fn: func(ctx context.Context, args map[string]any) (string, error) { return "ok", nil },
	})

	_, err := reg.Run(context.Background(), "corr", "session", "todo", map[string]any{"status": "invalid"})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "not in enum") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegistryRunShorthandSchemaTypeCheck(t *testing.T) {
	reg := New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	reg.Register(Tool{
		Name:   "read",
		Schema: map[string]any{"path": "string", "offset": "integer"},
		Fn:     func(ctx context.Context, args map[string]any) (string, error) { return "ok", nil },
	})

	_, err := reg.Run(context.Background(), "corr", "session", "read", map[string]any{"path": 123})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "expected string") {
		t.Fatalf("unexpected error: %v", err)
	}
}
