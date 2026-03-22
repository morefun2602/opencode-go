package providerstate

import (
	"context"
	"os"
	"sort"
	"strings"

	"github.com/morefun2602/opencode-go/internal/config"
)

type BuildOptions struct {
	DisableModelsFetch bool
	ModelsURL          string
}

func Build(ctx context.Context, cfg config.Config, opts BuildOptions) (State, error) {
	db, err := LoadModelsDev(ctx, cfg.DataDir, opts.DisableModelsFetch, opts.ModelsURL)
	if err != nil {
		// Degrade gracefully: continue with empty db, config/env may still provide providers.
		db = map[string]ProviderInfo{}
	}

	state := State{Providers: map[string]ActiveProvider{}}
	enabledSet := toSet(cfg.EnabledProviders)
	disabledSet := toSet(cfg.DisabledProviders)

	allowProvider := func(id string) bool {
		if len(enabledSet) > 0 {
			if !enabledSet[id] {
				return false
			}
		}
		if disabledSet[id] {
			return false
		}
		return true
	}

	// 1) Seed providers from models database by env/config/custom loader rules.
	for id, p := range db {
		if !allowProvider(id) {
			continue
		}

		ap := ActiveProvider{
			ID:      id,
			Name:    p.Name,
			Source:  SourceCustom,
			BaseURL: p.API,
			Type:    providerTypeForID(id),
		}

		cfgP, cfgHas := cfg.Providers[id]
		if cfgHas {
			if cfgP.APIKey != "" {
				ap.APIKey = cfgP.APIKey
				ap.Source = SourceConfig
			}
			if cfgP.BaseURL != "" {
				ap.BaseURL = cfgP.BaseURL
			}
			if cfgP.Type != "" {
				ap.Type = cfgP.Type
			}
		}

		for _, envName := range p.Env {
			if v := os.Getenv(envName); v != "" {
				ap.APIKey = v
				ap.Source = SourceEnv
				break
			}
		}

		models := modelIDsFromProvider(p)
		if id == "opencode" {
			// Original behavior: without key, keep only free models and use public key.
			if ap.APIKey == "" {
				models = freeModelIDsFromProvider(p)
				ap.APIKey = "public"
				ap.Source = SourceCustom
			}
			if ap.BaseURL == "" {
				ap.BaseURL = "https://opencode.ai/zen/v1"
			}
			if ap.Type == "" {
				ap.Type = "opencode"
			}
		} else if ap.APIKey == "" && !cfgHas {
			// Match upstream autoload semantics: most providers are not enabled
			// unless explicitly configured or backed by env/auth credentials.
			continue
		}

		models = filterModelStatus(models, p)
		if len(models) == 0 {
			continue
		}
		ap.Models = models
		state.Providers[id] = ap
	}

	// 2) Ensure config providers are included even if absent in models db.
	for id, cp := range cfg.Providers {
		if !allowProvider(id) {
			continue
		}
		ap, ok := state.Providers[id]
		if !ok {
			ap = ActiveProvider{
				ID:      id,
				Name:    id,
				Source:  SourceConfig,
				APIKey:  cp.APIKey,
				BaseURL: cp.BaseURL,
				Type:    cp.Type,
			}
			if ap.Type == "" {
				ap.Type = providerTypeForID(id)
			}
			if ap.Type == "" && ap.BaseURL != "" {
				ap.Type = "openai-compatible"
			}
			// Keep current behavior compatibility: at least one model must exist.
			if cp.Model != "" {
				ap.Models = []string{cp.Model}
			} else if id == "openai" {
				ap.Models = []string{"gpt-4o", "gpt-4o-mini", "gpt-4.1", "gpt-4.1-mini", "gpt-4.1-nano", "o3-mini"}
			} else if id == "anthropic" {
				ap.Models = []string{"claude-sonnet-4-20250514", "claude-3-5-sonnet-20241022", "claude-3-5-haiku-20241022"}
			} else if id == "opencode" {
				ap.Models = []string{"gpt-5-nano"}
			}
		} else {
			if cp.APIKey != "" {
				ap.APIKey = cp.APIKey
				ap.Source = SourceConfig
			}
			if cp.BaseURL != "" {
				ap.BaseURL = cp.BaseURL
			}
			if cp.Type != "" {
				ap.Type = cp.Type
			}
			if cp.Model != "" {
				ap.Models = []string{cp.Model}
			}
		}
		if len(ap.Models) == 0 {
			continue
		}
		state.Providers[id] = ap
	}

	state.Default = pickDefaultModel(cfg, state)
	state.Small = pickSmallModel(cfg, state)
	return state, nil
}

func providerTypeForID(id string) string {
	switch id {
	case "openai":
		return "openai"
	case "anthropic":
		return "anthropic"
	case "opencode":
		return "opencode"
	default:
		return "openai-compatible"
	}
}

func modelIDsFromProvider(p ProviderInfo) []string {
	ids := make([]string, 0, len(p.Models))
	for _, m := range p.Models {
		if m.ID != "" {
			ids = append(ids, m.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func freeModelIDsFromProvider(p ProviderInfo) []string {
	ids := make([]string, 0, len(p.Models))
	for _, m := range p.Models {
		if m.ID != "" && m.Cost.Input == 0 {
			ids = append(ids, m.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func filterModelStatus(ids []string, p ProviderInfo) []string {
	out := make([]string, 0, len(ids))
	enableExperimental := os.Getenv("OPENCODE_ENABLE_EXPERIMENTAL_MODELS") != ""
	for _, id := range ids {
		status := ""
		for _, m := range p.Models {
			if m.ID == id {
				status = strings.ToLower(m.Status)
				break
			}
		}
		if status == "deprecated" {
			continue
		}
		if status == "alpha" && !enableExperimental {
			continue
		}
		out = append(out, id)
	}
	return out
}

func pickDefaultModel(cfg config.Config, state State) string {
	if cfg.Model != "" {
		return cfg.Model
	}
	priority := []string{"gpt-5", "claude-sonnet-4", "big-pickle", "gemini-3-pro"}
	type candidate struct {
		ref      string
		priority int
	}
	best := candidate{priority: -1}
	for pid, p := range state.Providers {
		for _, m := range p.Models {
			ref := pid + "/" + m
			score := 0
			for i, key := range priority {
				if strings.Contains(m, key) {
					score = len(priority) - i
					break
				}
			}
			if score > best.priority || (score == best.priority && ref < best.ref) {
				best = candidate{ref: ref, priority: score}
			}
		}
	}
	return best.ref
}

func pickSmallModel(cfg config.Config, state State) string {
	if cfg.SmallModel != "" {
		return cfg.SmallModel
	}
	priority := []string{"claude-haiku-4-5", "claude-haiku-4.5", "3-5-haiku", "3.5-haiku", "gemini-3-flash", "gemini-2.5-flash", "gpt-5-nano"}
	for pid, p := range state.Providers {
		_ = pid
		for _, key := range priority {
			for _, m := range p.Models {
				if strings.Contains(m, key) {
					return p.ID + "/" + m
				}
			}
		}
	}
	return pickDefaultModel(cfg, state)
}

func toSet(items []string) map[string]bool {
	out := map[string]bool{}
	for _, it := range items {
		if strings.TrimSpace(it) == "" {
			continue
		}
		out[it] = true
	}
	return out
}

