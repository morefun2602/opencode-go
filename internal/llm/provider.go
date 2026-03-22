package llm

import (
	"context"
)

type Part struct {
	Type       string         `json:"type"` // "text" | "tool_call" | "tool_result"
	Text       string         `json:"text,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	ToolName   string         `json:"tool_name,omitempty"`
	Args       map[string]any `json:"args,omitempty"`
	Result     string         `json:"result,omitempty"`
	IsError    bool           `json:"is_error,omitempty"`
}

type Message struct {
	Role    string `json:"role"` // "system" | "user" | "assistant" | "tool"
	Content string `json:"content,omitempty"`
	Parts   []Part `json:"parts,omitempty"`
}

type ToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters,omitempty"` // JSON Schema
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

type Response struct {
	Message      Message `json:"message"`
	Usage        Usage   `json:"usage"`
	Model        string  `json:"model"`
	FinishReason string  `json:"finish_reason"` // "stop" | "tool_calls" | "length"
}

// Provider 语言模型出站调用抽象。
type Provider interface {
	Name() string
	Models() []string
	Chat(ctx context.Context, msgs []Message, tools []ToolDef) (*Response, error)
	ChatStream(ctx context.Context, msgs []Message, tools []ToolDef, chunk func(*Response) error) (*Response, error)
}

// ProviderWithModel is an optional extension that lets callers override
// the model per request while keeping provider instances reusable.
type ProviderWithModel interface {
	Provider
	ChatWithModel(ctx context.Context, model string, msgs []Message, tools []ToolDef) (*Response, error)
	ChatStreamWithModel(ctx context.Context, model string, msgs []Message, tools []ToolDef, chunk func(*Response) error) (*Response, error)
}
