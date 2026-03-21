package tool

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/morefun2602/opencode-go/internal/tools"
)

func TestLs_BasicDirectory(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "src"), 0o755)
	os.WriteFile(filepath.Join(dir, "src", "main.go"), []byte("package main"), 0o644)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# readme"), 0o644)

	reg := tools.New(nil)
	registerLs(reg, dir)

	result, err := reg.Run(context.Background(), "", "", "ls", map[string]any{
		"path": ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "README.md") {
		t.Errorf("expected README.md in output, got: %s", result)
	}
	if !strings.Contains(result, "src") {
		t.Errorf("expected src in output, got: %s", result)
	}
}

func TestLs_GitignoreRespected(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\nbuild/"), 0o644)
	os.WriteFile(filepath.Join(dir, "app.go"), []byte("package main"), 0o644)
	os.WriteFile(filepath.Join(dir, "debug.log"), []byte("log"), 0o644)
	os.MkdirAll(filepath.Join(dir, "build"), 0o755)
	os.WriteFile(filepath.Join(dir, "build", "out"), []byte("bin"), 0o644)

	reg := tools.New(nil)
	registerLs(reg, dir)

	result, err := reg.Run(context.Background(), "", "", "ls", map[string]any{
		"path": ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result, "debug.log") {
		t.Errorf("should not contain ignored file, got: %s", result)
	}
	if strings.Contains(result, "build") {
		t.Errorf("should not contain ignored dir, got: %s", result)
	}
}

func TestLs_PathOutsideWorkspace(t *testing.T) {
	dir := t.TempDir()

	reg := tools.New(nil)
	registerLs(reg, dir)

	_, err := reg.Run(context.Background(), "", "", "ls", map[string]any{
		"path": "/etc",
	})
	if err == nil {
		t.Fatal("expected error for path outside workspace")
	}
}
