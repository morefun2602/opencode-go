package tool

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/morefun2602/opencode-go/internal/tools"
)

func TestMultiedit_Success(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	os.WriteFile(f, []byte("hello world\nfoo bar\nbaz qux"), 0o644)

	reg := tools.New(nil)
	registerMultiedit(reg, dir)

	result, err := reg.Run(context.Background(), "", "", "multiedit", map[string]any{
		"path": "test.txt",
		"edits": []any{
			map[string]any{"old_string": "hello", "new_string": "hi"},
			map[string]any{"old_string": "foo", "new_string": "updated"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result != "ok: 2 edits applied" {
		t.Errorf("unexpected result: %s", result)
	}

	content, _ := os.ReadFile(f)
	got := string(content)
	if got != "hi world\nupdated bar\nbaz qux" {
		t.Errorf("unexpected file content: %s", got)
	}
}

func TestMultiedit_NotFound(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	os.WriteFile(f, []byte("hello world"), 0o644)

	reg := tools.New(nil)
	registerMultiedit(reg, dir)

	_, err := reg.Run(context.Background(), "", "", "multiedit", map[string]any{
		"path": "test.txt",
		"edits": []any{
			map[string]any{"old_string": "nonexistent", "new_string": "replaced"},
		},
	})
	if err == nil {
		t.Fatal("expected error for missing old_string")
	}
}

func TestMultiedit_NotUnique(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	os.WriteFile(f, []byte("hello hello hello"), 0o644)

	reg := tools.New(nil)
	registerMultiedit(reg, dir)

	_, err := reg.Run(context.Background(), "", "", "multiedit", map[string]any{
		"path": "test.txt",
		"edits": []any{
			map[string]any{"old_string": "hello", "new_string": "replaced"},
		},
	})
	if err == nil {
		t.Fatal("expected error for non-unique old_string")
	}
}
