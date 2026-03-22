package providerstate

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadModelsDev_FallbackToSnapshot(t *testing.T) {
	providers, err := LoadModelsDev(context.Background(), t.TempDir(), true, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(providers) == 0 {
		t.Fatal("expected non-empty providers from snapshot")
	}
	if _, ok := providers["opencode"]; !ok {
		t.Fatal("expected opencode provider in snapshot")
	}
}

func TestLoadModelsDev_FromCache(t *testing.T) {
	dataDir := t.TempDir()
	cache := filepath.Join(dataDir, cacheRelPath)
	if err := os.MkdirAll(filepath.Dir(cache), 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `{"p":{"id":"p","name":"P","env":["P_KEY"],"models":{"m":{"id":"m","name":"M"}}}}`
	if err := os.WriteFile(cache, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	providers, err := LoadModelsDev(context.Background(), dataDir, true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := providers["p"]; !ok {
		t.Fatalf("expected provider loaded from cache")
	}
}

