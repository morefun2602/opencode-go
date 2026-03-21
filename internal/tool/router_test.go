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
