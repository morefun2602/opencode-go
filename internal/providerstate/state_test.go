package providerstate

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/morefun2602/opencode-go/internal/config"
)

func TestBuild_NoKeys_EnablesOpencodeFree(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENCODE_API_KEY", "")
	t.Setenv("OPENCODE_DISABLE_MODELS_FETCH", "1")

	cfg := config.Defaults()
	cfg.DataDir = t.TempDir()

	st, err := Build(context.Background(), cfg, BuildOptions{DisableModelsFetch: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := st.Providers["opencode"]; !ok {
		t.Fatalf("expected opencode provider to be enabled without key")
	}
	if _, ok := st.Providers["openai"]; ok {
		t.Fatalf("openai should not auto-enable without key/config")
	}
	if st.Default == "" {
		t.Fatalf("default model should be selected")
	}
}

func TestBuild_OpenAIKey_EnablesOpenAI(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "dummy")
	t.Setenv("OPENCODE_DISABLE_MODELS_FETCH", "1")

	cfg := config.Defaults()
	cfg.DataDir = t.TempDir()
	st, err := Build(context.Background(), cfg, BuildOptions{DisableModelsFetch: true})
	if err != nil {
		t.Fatal(err)
	}
	if p, ok := st.Providers["openai"]; !ok || len(p.Models) == 0 {
		t.Fatalf("expected openai provider with models, got %#v", p)
	}
}

func TestBuild_EnabledDisabledProviders(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "dummy")
	t.Setenv("OPENCODE_DISABLE_MODELS_FETCH", "1")

	cfg := config.Defaults()
	cfg.DataDir = t.TempDir()
	cfg.EnabledProviders = []string{"openai"}
	cfg.DisabledProviders = []string{"opencode"}

	st, err := Build(context.Background(), cfg, BuildOptions{DisableModelsFetch: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := st.Providers["openai"]; !ok {
		t.Fatal("openai should be enabled")
	}
	if _, ok := st.Providers["opencode"]; ok {
		t.Fatal("opencode should be disabled")
	}
}

func TestBuild_UsesCacheWhenPresent(t *testing.T) {
	cfg := config.Defaults()
	cfg.DataDir = t.TempDir()

	cachePath := filepath.Join(cfg.DataDir, cacheRelPath)
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `{"demo":{"id":"demo","name":"Demo","env":[],"models":{"m1":{"id":"m1","name":"M1"}}}}`
	if err := os.WriteFile(cachePath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg.Providers = map[string]config.InternalProvider{
		"demo": {BaseURL: "https://example.com/v1", Type: "openai-compatible"},
	}
	st, err := Build(context.Background(), cfg, BuildOptions{DisableModelsFetch: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := st.Providers["demo"]; !ok {
		t.Fatal("expected demo provider from cache+config merge")
	}
}

