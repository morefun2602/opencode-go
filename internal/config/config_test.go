package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvOverridesFile(t *testing.T) {
	t.Setenv("OPENCODE_SERVER_LISTEN", "127.0.0.1:7777")
	t.Setenv("OPENCODE_CONFIG", "")
	dir := t.TempDir()
	p := filepath.Join(dir, "opencode.json")
	if err := os.WriteFile(p, []byte(`{"server":{"listen":"127.0.0.1:1111"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p, nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.Listen != "127.0.0.1:7777" {
		t.Fatalf("want env to override file, got %q", c.Listen)
	}
}

func TestFlagOverlay(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("OPENCODE_SERVER_LISTEN", "")
	t.Setenv("OPENCODE_CONFIG", "")
	c, err := Load("", &Config{Listen: "127.0.0.1:9999"})
	if err != nil {
		t.Fatal(err)
	}
	if c.Listen != "127.0.0.1:9999" {
		t.Fatalf("got %q", c.Listen)
	}
}

func TestNonLoopbackRequiresToken(t *testing.T) {
	t.Chdir(t.TempDir())
	_, err := Load("", &Config{Listen: "0.0.0.0:8080", AuthToken: ""})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDoomLoopWindowFromFileAndEnv(t *testing.T) {
	t.Setenv("OPENCODE_CONFIG", "")
	dir := t.TempDir()
	p := filepath.Join(dir, "opencode.json")
	if err := os.WriteFile(p, []byte(`{"doom_loop_window":5}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(p, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DoomLoopWindow != 5 {
		t.Fatalf("expected doom_loop_window=5 from file, got %d", cfg.DoomLoopWindow)
	}
	t.Setenv("OPENCODE_DOOM_LOOP_WINDOW", "7")
	cfg, err = Load(p, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DoomLoopWindow != 7 {
		t.Fatalf("expected env override doom_loop_window=7, got %d", cfg.DoomLoopWindow)
	}
}
