package tool

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/morefun2602/opencode-go/internal/policy"
	"github.com/morefun2602/opencode-go/internal/tools"
	"io"
	"log/slog"
)

func TestEditUniqueMatch(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	os.WriteFile(f, []byte("hello world"), 0o644)

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	reg := tools.New(log)
	registerEdit(reg, dir)
	result, err := reg.Run(context.Background(), "", "", "edit", map[string]any{
		"path": "test.txt", "old_string": "hello", "new_string": "goodbye",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result != "ok" {
		t.Fatalf("want 'ok', got %q", result)
	}
	b, _ := os.ReadFile(f)
	if string(b) != "goodbye world" {
		t.Fatalf("want 'goodbye world', got %q", string(b))
	}
}

func TestEditNotFound(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello"), 0o644)

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	reg := tools.New(log)
	registerEdit(reg, dir)
	_, err := reg.Run(context.Background(), "", "", "edit", map[string]any{
		"path": "test.txt", "old_string": "xyz", "new_string": "abc",
	})
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestEditMultipleMatches(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("aa aa aa"), 0o644)

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	reg := tools.New(log)
	registerEdit(reg, dir)
	_, err := reg.Run(context.Background(), "", "", "edit", map[string]any{
		"path": "test.txt", "old_string": "aa", "new_string": "bb",
	})
	if err == nil {
		t.Fatal("expected error for multiple matches")
	}
}

func TestEditPathOutside(t *testing.T) {
	dir := t.TempDir()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	pol := &policy.Policy{WorkspaceRoot: dir}
	_ = pol
	reg := tools.New(log)
	registerEdit(reg, dir)
	_, err := reg.Run(context.Background(), "", "", "edit", map[string]any{
		"path": "/etc/passwd", "old_string": "x", "new_string": "y",
	})
	if err == nil {
		t.Fatal("expected error for path outside workspace")
	}
}
