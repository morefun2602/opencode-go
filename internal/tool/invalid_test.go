package tool

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/morefun2602/opencode-go/internal/tools"
)

func TestInvalidTool(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	reg := tools.New(log)
	registerInvalid(reg)

	result, err := reg.Run(context.Background(), "", "", "invalid", map[string]any{
		"tool":  "unknown_tool",
		"error": "tool not found",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "unknown_tool") {
		t.Error("result should mention the original tool name")
	}
	if !strings.Contains(result, "invalid") {
		t.Error("result should describe the error")
	}
}
