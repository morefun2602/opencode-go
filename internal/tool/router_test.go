package tool

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/morefun2602/opencode-go/internal/tools"
)

func TestRouterUnknownTool(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	reg := tools.New(log)
	r := &Router{Builtin: reg, Log: log}
	_, err := r.Run(context.Background(), "c", "s", "nope", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	var eu *ErrUnknown
	if !errors.As(err, &eu) {
		t.Fatalf("want *ErrUnknown, got %v", err)
	}
}

func TestRouterCaseInsensitiveRepair(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	reg := tools.New(log)
	reg.Register(tools.Tool{
		Name: "read",
		Fn:   func(ctx context.Context, args map[string]any) (string, error) { return "ok", nil },
	})
	r := &Router{Builtin: reg, Log: log}

	out, err := r.Run(context.Background(), "c", "s", "Read", nil)
	if err != nil {
		t.Fatalf("expected case repair to work, got error: %v", err)
	}
	if out != "ok" {
		t.Fatalf("want 'ok', got %q", out)
	}
}

func TestRouterCaseInsensitiveStillUnknown(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	reg := tools.New(log)
	reg.Register(tools.Tool{
		Name: "read",
		Fn:   func(ctx context.Context, args map[string]any) (string, error) { return "ok", nil },
	})
	r := &Router{Builtin: reg, Log: log}

	_, err := r.Run(context.Background(), "c", "s", "NOPE", nil)
	if err == nil {
		t.Fatal("expected error for truly unknown tool")
	}
	var eu *ErrUnknown
	if !errors.As(err, &eu) {
		t.Fatalf("want *ErrUnknown, got %v", err)
	}
}
