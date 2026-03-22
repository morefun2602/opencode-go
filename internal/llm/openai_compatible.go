package llm

import "context"

type OpenAICompatible struct {
	inner  *OpenAI
	name   string
	models []string
}

func NewOpenAICompatible(name string, cfg OpenAIConfig) *OpenAICompatible {
	return &OpenAICompatible{inner: NewOpenAI(cfg), name: name, models: []string{cfg.Model}}
}

// NewOpenAICompatibleWithModels creates a compatible provider with an explicit model list.
func NewOpenAICompatibleWithModels(name string, cfg OpenAIConfig, models []string) *OpenAICompatible {
	return &OpenAICompatible{inner: NewOpenAI(cfg), name: name, models: models}
}

func (c *OpenAICompatible) Name() string { return c.name }
func (c *OpenAICompatible) Models() []string {
	if len(c.models) > 0 {
		return c.models
	}
	return []string{c.inner.model}
}

func (c *OpenAICompatible) Chat(ctx context.Context, msgs []Message, tools []ToolDef) (*Response, error) {
	return c.inner.Chat(ctx, msgs, tools)
}

func (c *OpenAICompatible) ChatStream(ctx context.Context, msgs []Message, tools []ToolDef, chunk func(*Response) error) (*Response, error) {
	return c.inner.ChatStream(ctx, msgs, tools, chunk)
}

var _ Provider = (*OpenAICompatible)(nil)
