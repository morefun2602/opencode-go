package skill

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDiscovery_Pull(t *testing.T) {
	mux := http.NewServeMux()
	idx := Index{
		Skills: []IndexSkill{
			{Name: "alpha", Files: []string{"SKILL.md", "reference.md"}},
			{Name: "beta", Files: []string{"SKILL.md"}},
		},
	}
	idxJSON, _ := json.Marshal(idx)
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		w.Write(idxJSON)
	})
	mux.HandleFunc("/alpha/SKILL.md", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("---\nname: alpha\ndescription: Alpha skill\n---\nAlpha body"))
	})
	mux.HandleFunc("/alpha/reference.md", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("# Reference"))
	})
	mux.HandleFunc("/beta/SKILL.md", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("---\nname: beta\ndescription: Beta skill\n---\nBeta body"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cacheDir := t.TempDir()
	d := NewDiscovery(cacheDir, slog.Default())
	dirs := d.Pull(srv.URL)

	if len(dirs) != 2 {
		t.Fatalf("expected 2 dirs, got %d", len(dirs))
	}

	alphaSkill := filepath.Join(cacheDir, "skills", "alpha", "SKILL.md")
	if _, err := os.Stat(alphaSkill); err != nil {
		t.Fatalf("alpha SKILL.md not cached: %v", err)
	}
	alphaRef := filepath.Join(cacheDir, "skills", "alpha", "reference.md")
	if _, err := os.Stat(alphaRef); err != nil {
		t.Fatalf("alpha reference.md not cached: %v", err)
	}
}

func TestDiscovery_Pull_IndexUnreachable(t *testing.T) {
	cacheDir := t.TempDir()
	d := NewDiscovery(cacheDir, slog.Default())
	dirs := d.Pull("http://127.0.0.1:1/nonexistent")

	if len(dirs) != 0 {
		t.Fatalf("expected 0 dirs for unreachable URL, got %d", len(dirs))
	}
}

func TestDiscovery_Pull_MissingSkillMD(t *testing.T) {
	mux := http.NewServeMux()
	idx := Index{
		Skills: []IndexSkill{
			{Name: "no-skill", Files: []string{"README.md"}},
		},
	}
	idxJSON, _ := json.Marshal(idx)
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		w.Write(idxJSON)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cacheDir := t.TempDir()
	d := NewDiscovery(cacheDir, slog.Default())
	dirs := d.Pull(srv.URL)

	if len(dirs) != 0 {
		t.Fatalf("expected 0 dirs (no SKILL.md in index), got %d", len(dirs))
	}
}

func TestDiscovery_Pull_CacheHit(t *testing.T) {
	downloadCount := 0
	mux := http.NewServeMux()
	idx := Index{
		Skills: []IndexSkill{
			{Name: "cached", Files: []string{"SKILL.md"}},
		},
	}
	idxJSON, _ := json.Marshal(idx)
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		w.Write(idxJSON)
	})
	mux.HandleFunc("/cached/SKILL.md", func(w http.ResponseWriter, r *http.Request) {
		downloadCount++
		w.Write([]byte("# Cached Skill"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cacheDir := t.TempDir()
	d := NewDiscovery(cacheDir, slog.Default())

	// First pull
	d.Pull(srv.URL)
	if downloadCount != 1 {
		t.Fatalf("expected 1 download, got %d", downloadCount)
	}

	// Second pull should skip download
	d.Pull(srv.URL)
	if downloadCount != 1 {
		t.Fatalf("expected still 1 download (cached), got %d", downloadCount)
	}
}
