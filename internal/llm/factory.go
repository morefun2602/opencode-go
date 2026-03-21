package llm

import "os"

type ProviderConfig struct {
	APIKey  string
	BaseURL string
	Model   string
	Type    string
}

func NewProvider(name string, cfg ProviderConfig) Provider {
	typ := cfg.Type
	if typ == "" {
		typ = name
	}
	switch typ {
	case "openai":
		if cfg.APIKey == "" {
			cfg.APIKey = os.Getenv("OPENAI_API_KEY")
		}
		return NewOpenAI(OpenAIConfig{APIKey: cfg.APIKey, BaseURL: cfg.BaseURL, Model: cfg.Model})
	case "anthropic":
		if cfg.APIKey == "" {
			cfg.APIKey = os.Getenv("ANTHROPIC_API_KEY")
		}
		return NewAnthropic(AnthropicConfig{APIKey: cfg.APIKey, Model: cfg.Model})
	case "openai-compatible":
		if cfg.APIKey == "" {
			cfg.APIKey = os.Getenv("OPENAI_API_KEY")
		}
		return NewOpenAICompatible(name, OpenAIConfig{APIKey: cfg.APIKey, BaseURL: cfg.BaseURL, Model: cfg.Model})
	default:
		if cfg.BaseURL != "" {
			return NewOpenAICompatible(name, OpenAIConfig{APIKey: cfg.APIKey, BaseURL: cfg.BaseURL, Model: cfg.Model})
		}
		return Stub{}
	}
}
