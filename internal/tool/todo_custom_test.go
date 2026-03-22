package tool

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/morefun2602/opencode-go/internal/policy"
	"github.com/morefun2602/opencode-go/internal/tools"
)

func TestTodoreadAfterTodowrite(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	reg := tools.New(log)
	RegisterBuiltin(reg, &policy.Policy{WorkspaceRoot: t.TempDir()}, nil, nil)

	_, err := reg.Run(context.Background(), "corr", "s1", "todowrite", map[string]any{
		"merge": true,
		"todos": []any{
			map[string]any{"id": "t1", "content": "hello", "status": "in_progress"},
		},
	})
	if err != nil {
		t.Fatalf("todowrite failed: %v", err)
	}

	out, err := reg.Run(context.Background(), "corr", "s1", "todoread", map[string]any{})
	if err != nil {
		t.Fatalf("todoread failed: %v", err)
	}
	if !strings.Contains(out, "t1") {
		t.Fatalf("expected todo content, got: %q", out)
	}
}

func TestRegisterCustomToolsFromWorkspace(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	root := t.TempDir()
	customDir := filepath.Join(root, ".opencode", "tools")
	if err := os.MkdirAll(customDir, 0o755); err != nil {
		t.Fatal(err)
	}
	def := `{
  "name": "echo_json",
  "description": "echo stdin json",
  "command": "read payload; echo $payload",
  "tags": ["execute"],
  "schema": {
    "type": "object",
    "properties": {
      "msg": { "type": "string" }
    },
    "required": ["msg"]
  }
}`
	if err := os.WriteFile(filepath.Join(customDir, "echo_json.json"), []byte(def), 0o644); err != nil {
		t.Fatal(err)
	}

	reg := tools.New(log)
	RegisterCustomToolsFromWorkspace(reg, root, log)
	if !reg.Has("echo_json") {
		t.Fatal("expected custom tool to be registered")
	}

	out, err := reg.Run(context.Background(), "corr", "s1", "echo_json", map[string]any{"msg": "hello"})
	if err != nil {
		t.Fatalf("custom tool run failed: %v", err)
	}
	if !strings.Contains(out, "hello") {
		t.Fatalf("unexpected custom tool output: %q", out)
	}
}
