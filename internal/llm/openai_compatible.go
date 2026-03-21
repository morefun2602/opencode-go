package llm

import "context"

type OpenAICompatible struct {
	inner *OpenAI
	name  string
}

func NewOpenAICompatible(name string, cfg OpenAIConfig) *OpenAICompatible {
	return &OpenAICompatible{inner: NewOpenAI(cfg), name: name}
}

func (c *OpenAICompatible) Name() string     { return c.name }
func (c *OpenAICompatible) Models() []string  { return []string{c.inner.model} }

func (c *OpenAICompatible) Chat(ctx context.Context, msgs []Message, tools []ToolDef) (*Response, error) {
	return c.inner.Chat(ctx, msgs, tools)
}

func (c *OpenAICompatible) ChatStream(ctx context.Context, msgs []Message, tools []ToolDef, chunk func(*Response) error) (*Response, error) {
	return c.inner.ChatStream(ctx, msgs, tools, chunk)
}

var _ Provider = (*OpenAICompatible)(nil)
