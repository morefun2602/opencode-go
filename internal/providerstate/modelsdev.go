package providerstate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	defaultModelsURL = "https://models.dev/api.json"
	cacheRelPath     = "cache/models.json"
)

// Minimal snapshot fallback used when cache+network are unavailable.
var snapshot = map[string]ProviderInfo{
	"opencode": {
		ID:   "opencode",
		Name: "OpenCode",
		Env:  []string{"OPENCODE_API_KEY"},
		API:  "https://opencode.ai/zen/v1",
		NPM:  "@ai-sdk/openai-compatible",
		Models: map[string]ModelInfo{
			"gpt-5-nano":          {ID: "gpt-5-nano", Name: "gpt-5-nano", Cost: ModelCost{Input: 0}},
			"grok-code":           {ID: "grok-code", Name: "grok-code", Cost: ModelCost{Input: 0}},
			"glm-5-free":          {ID: "glm-5-free", Name: "glm-5-free", Cost: ModelCost{Input: 0}},
			"kimi-k2.5-free":      {ID: "kimi-k2.5-free", Name: "kimi-k2.5-free", Cost: ModelCost{Input: 0}},
			"nemotron-3-super-free": {ID: "nemotron-3-super-free", Name: "nemotron-3-super-free", Cost: ModelCost{Input: 0}},
		},
	},
	"openai": {
		ID:   "openai",
		Name: "OpenAI",
		Env:  []string{"OPENAI_API_KEY"},
		NPM:  "@ai-sdk/openai",
		Models: map[string]ModelInfo{
			"gpt-4o":       {ID: "gpt-4o", Name: "gpt-4o"},
			"gpt-4o-mini":  {ID: "gpt-4o-mini", Name: "gpt-4o-mini"},
			"gpt-4.1":      {ID: "gpt-4.1", Name: "gpt-4.1"},
			"gpt-4.1-mini": {ID: "gpt-4.1-mini", Name: "gpt-4.1-mini"},
			"gpt-5-nano":   {ID: "gpt-5-nano", Name: "gpt-5-nano"},
		},
	},
	"anthropic": {
		ID:   "anthropic",
		Name: "Anthropic",
		Env:  []string{"ANTHROPIC_API_KEY"},
		NPM:  "@ai-sdk/anthropic",
		Models: map[string]ModelInfo{
			"claude-sonnet-4-20250514":      {ID: "claude-sonnet-4-20250514", Name: "claude-sonnet-4-20250514"},
			"claude-3-5-sonnet-20241022":    {ID: "claude-3-5-sonnet-20241022", Name: "claude-3-5-sonnet-20241022"},
			"claude-3-5-haiku-20241022":     {ID: "claude-3-5-haiku-20241022", Name: "claude-3-5-haiku-20241022"},
		},
	},
}

func LoadModelsDev(ctx context.Context, dataDir string, disableFetch bool, modelsURL string) (map[string]ProviderInfo, error) {
	if modelsURL == "" {
		modelsURL = defaultModelsURL
	}
	cachePath := filepath.Join(dataDir, cacheRelPath)

	// 1) cache
	if providers, err := readProvidersJSON(cachePath); err == nil && len(providers) > 0 {
		// best effort async refresh behavior (synchronous call-site, fire-and-forget)
		if !disableFetch {
			go func() {
				_, _ = refreshModelsDev(context.Background(), cachePath, modelsURL)
			}()
		}
		return providers, nil
	}

	// 2) embedded snapshot
	providers := cloneProviders(snapshot)
	if len(providers) > 0 {
		if !disableFetch {
			go func() {
				_, _ = refreshModelsDev(context.Background(), cachePath, modelsURL)
			}()
		}
		return providers, nil
	}

	// 3) network direct
	if disableFetch {
		return map[string]ProviderInfo{}, nil
	}
	providers, err := refreshModelsDev(ctx, cachePath, modelsURL)
	if err != nil {
		return map[string]ProviderInfo{}, err
	}
	return providers, nil
}

func refreshModelsDev(ctx context.Context, cachePath, modelsURL string) (map[string]ProviderInfo, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, modelsURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("models.dev status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	providers, err := parseProvidersJSON(body)
	if err != nil {
		return nil, err
	}

	_ = os.MkdirAll(filepath.Dir(cachePath), 0o755)
	_ = os.WriteFile(cachePath, body, 0o644)
	return providers, nil
}

func readProvidersJSON(path string) (map[string]ProviderInfo, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseProvidersJSON(b)
}

func parseProvidersJSON(b []byte) (map[string]ProviderInfo, error) {
	// Loose parsing to avoid hard failures when schema evolves.
	type modelCost struct {
		Input float64 `json:"input"`
	}
	type rawModel struct {
		ID     string    `json:"id"`
		Name   string    `json:"name"`
		Status string    `json:"status"`
		Cost   modelCost `json:"cost"`
	}
	type rawProvider struct {
		ID     string              `json:"id"`
		Name   string              `json:"name"`
		Env    []string            `json:"env"`
		API    string              `json:"api"`
		NPM    string              `json:"npm"`
		Models map[string]rawModel `json:"models"`
	}
	raw := map[string]rawProvider{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	out := make(map[string]ProviderInfo, len(raw))
	for key, rp := range raw {
		id := rp.ID
		if id == "" {
			id = key
		}
		pi := ProviderInfo{
			ID:     id,
			Name:   rp.Name,
			Env:    rp.Env,
			API:    rp.API,
			NPM:    rp.NPM,
			Models: map[string]ModelInfo{},
		}
		if pi.Name == "" {
			pi.Name = id
		}
		for mk, rm := range rp.Models {
			mid := rm.ID
			if mid == "" {
				mid = mk
			}
			pi.Models[mk] = ModelInfo{
				ID:     mid,
				Name:   rm.Name,
				Status: rm.Status,
				Cost:   ModelCost{Input: rm.Cost.Input},
			}
		}
		out[id] = pi
	}
	return out, nil
}

func cloneProviders(in map[string]ProviderInfo) map[string]ProviderInfo {
	out := make(map[string]ProviderInfo, len(in))
	for k, p := range in {
		cp := p
		cp.Env = append([]string(nil), p.Env...)
		cp.Models = map[string]ModelInfo{}
		for mk, mv := range p.Models {
			cp.Models[mk] = mv
		}
		out[k] = cp
	}
	return out
}

