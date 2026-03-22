package llm

import (
	"context"
	"testing"
)

func TestParseModel(t *testing.T) {
	tests := []struct {
		input    string
		wantProv string
		wantMod  string
	}{
		{"anthropic/claude-3-5-sonnet", "anthropic", "claude-3-5-sonnet"},
		{"gpt-4o", "", "gpt-4o"},
		{"", "", ""},
		{"openai/gpt-4o-mini", "openai", "gpt-4o-mini"},
	}

	for _, tt := range tests {
		ref := ParseModel(tt.input)
		if ref.ProviderID != tt.wantProv {
			t.Errorf("ParseModel(%q).ProviderID = %q, want %q", tt.input, ref.ProviderID, tt.wantProv)
		}
		if ref.ModelID != tt.wantMod {
			t.Errorf("ParseModel(%q).ModelID = %q, want %q", tt.input, ref.ModelID, tt.wantMod)
		}
	}
}

type fakeRouterProvider struct {
	name   string
	models []string
}

func (f *fakeRouterProvider) Name() string { return f.name }
func (f *fakeRouterProvider) Models() []string {
	return f.models
}
func (f *fakeRouterProvider) Chat(_ context.Context, _ []Message, _ []ToolDef) (*Response, error) {
	return nil, nil
}
func (f *fakeRouterProvider) ChatStream(_ context.Context, _ []Message, _ []ToolDef, _ func(*Response) error) (*Response, error) {
	return nil, nil
}

func TestRouterDefaultModel(t *testing.T) {
	reg := NewRegistry()
	reg.Register("openai", func() Provider {
		return Stub{}
	})

	r := NewRouter(reg, "openai/gpt-4o", "")
	ref := r.DefaultModel()
	if ref.ProviderID != "openai" || ref.ModelID != "gpt-4o" {
		t.Errorf("DefaultModel = %+v, want openai/gpt-4o", ref)
	}
}

func TestRouterDefaultModelFallback(t *testing.T) {
	reg := NewRegistry()
	reg.Register("openai", func() Provider {
		return Stub{}
	})

	r := NewRouter(reg, "", "")
	ref := r.DefaultModel()
	if ref.ProviderID != "openai" {
		t.Errorf("DefaultModel.ProviderID = %q, want openai", ref.ProviderID)
	}
}

func TestRouterResolve(t *testing.T) {
	reg := NewRegistry()
	reg.Register("openai", func() Provider {
		return Stub{}
	})

	r := NewRouter(reg, "openai/gpt-4o", "")
	prov, model, err := r.Resolve(ModelRef{ProviderID: "openai", ModelID: "gpt-4o"})
	if err != nil {
		t.Fatal(err)
	}
	if prov == nil {
		t.Fatal("provider should not be nil")
	}
	_ = model
}

func TestRouterSetDefaultAffectsResolveDefault(t *testing.T) {
	reg := NewRegistry()
	reg.Register("openai", func() Provider {
		return &fakeRouterProvider{name: "openai", models: []string{"gpt-4o", "gpt-5-nano"}}
	})

	r := NewRouter(reg, "openai/gpt-4o", "")
	r.SetDefault(ParseModel("openai/gpt-5-nano"))
	_, model, err := r.ResolveDefault()
	if err != nil {
		t.Fatal(err)
	}
	if model != "gpt-5-nano" {
		t.Fatalf("want gpt-5-nano, got %q", model)
	}
}
