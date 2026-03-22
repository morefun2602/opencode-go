package llm

import "os"

type ProviderConfig struct {
	APIKey  string
	BaseURL string
	Model   string
	Models  []string
	Type    string
}

// OpenCode Zen free models (cost.input == 0 on models.dev).
var opencodeDefaultModels = []string{
	"gpt-5-nano",
	"grok-code",
	"glm-4.7-free",
	"glm-5-free",
	"kimi-k2.5-free",
	"nemotron-3-super-free",
	"mimo-v2-flash-free",
	"mimo-v2-omni-free",
	"mimo-v2-pro-free",
	"minimax-m2.1-free",
	"minimax-m2.5-free",
	"big-pickle",
	"trinity-large-preview-free",
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
	case "opencode":
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://opencode.ai/zen/v1"
		}
		if cfg.APIKey == "" {
			cfg.APIKey = os.Getenv("OPENCODE_API_KEY")
		}
		if cfg.APIKey == "" {
			cfg.APIKey = "public"
		}
		model := cfg.Model
		if model == "" {
			if len(cfg.Models) > 0 {
				model = cfg.Models[0]
			} else {
				model = opencodeDefaultModels[0]
			}
		}
		models := opencodeDefaultModels
		if len(cfg.Models) > 0 {
			models = cfg.Models
		}
		return NewOpenAICompatibleWithModels(name, OpenAIConfig{
			APIKey: cfg.APIKey, BaseURL: cfg.BaseURL, Model: model,
		}, models)
	case "openai-compatible":
		if cfg.APIKey == "" {
			cfg.APIKey = os.Getenv("OPENAI_API_KEY")
		}
		if len(cfg.Models) > 0 {
			return NewOpenAICompatibleWithModels(name, OpenAIConfig{APIKey: cfg.APIKey, BaseURL: cfg.BaseURL, Model: cfg.Model}, cfg.Models)
		}
		return NewOpenAICompatible(name, OpenAIConfig{APIKey: cfg.APIKey, BaseURL: cfg.BaseURL, Model: cfg.Model})
	default:
		if cfg.BaseURL != "" {
			if len(cfg.Models) > 0 {
				return NewOpenAICompatibleWithModels(name, OpenAIConfig{APIKey: cfg.APIKey, BaseURL: cfg.BaseURL, Model: cfg.Model}, cfg.Models)
			}
			return NewOpenAICompatible(name, OpenAIConfig{APIKey: cfg.APIKey, BaseURL: cfg.BaseURL, Model: cfg.Model})
		}
		return Stub{}
	}
}
