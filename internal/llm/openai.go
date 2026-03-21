package llm

import (
	"context"
	"encoding/json"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type OpenAI struct {
	client openai.Client
	model  string
}

type OpenAIConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

func NewOpenAI(cfg OpenAIConfig) *OpenAI {
	var opts []option.RequestOption
	if cfg.APIKey != "" {
		opts = append(opts, option.WithAPIKey(cfg.APIKey))
	}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}
	model := cfg.Model
	if model == "" {
		model = "gpt-4o"
	}
	return &OpenAI{client: openai.NewClient(opts...), model: model}
}

func (o *OpenAI) Name() string { return "openai" }

func (o *OpenAI) Models() []string {
	return []string{"gpt-4o", "gpt-4o-mini", "gpt-4.1", "gpt-4.1-mini", "gpt-4.1-nano", "o3-mini"}
}

func (o *OpenAI) Chat(ctx context.Context, msgs []Message, tools []ToolDef) (*Response, error) {
	params := o.params(msgs, tools)
	comp, err := o.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, err
	}
	return o.mapResponse(comp), nil
}

func (o *OpenAI) ChatStream(ctx context.Context, msgs []Message, tools []ToolDef, chunk func(*Response) error) (*Response, error) {
	params := o.params(msgs, tools)
	stream := o.client.Chat.Completions.NewStreaming(ctx, params)
	acc := openai.ChatCompletionAccumulator{}
	for stream.Next() {
		delta := stream.Current()
		acc.AddChunk(delta)
		partial := &Response{Model: string(delta.Model), FinishReason: "stop"}
		for _, ch := range delta.Choices {
			if ch.Delta.Content != "" {
				partial.Message.Role = "assistant"
				partial.Message.Content = ch.Delta.Content
			}
		}
		if err := chunk(partial); err != nil {
			return nil, err
		}
	}
	if err := stream.Err(); err != nil {
		return nil, err
	}
	if len(acc.Choices) == 0 {
		return &Response{Model: o.model, FinishReason: "stop", Message: Message{Role: "assistant"}}, nil
	}
	return o.mapChoice(acc.Choices[0], string(acc.Model)), nil
}

func (o *OpenAI) params(msgs []Message, tools []ToolDef) openai.ChatCompletionNewParams {
	p := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(o.model),
		Messages: o.mapMessages(msgs),
	}
	if len(tools) > 0 {
		td := make([]openai.ChatCompletionToolParam, 0, len(tools))
		for _, t := range tools {
			schema := openai.FunctionParameters(t.Parameters)
			td = append(td, openai.ChatCompletionToolParam{
				Function: openai.FunctionDefinitionParam{
					Name:        t.Name,
					Description: openai.String(t.Description),
					Parameters:  schema,
				},
			})
		}
		p.Tools = td
	}
	return p
}

func (o *OpenAI) mapMessages(msgs []Message) []openai.ChatCompletionMessageParamUnion {
	out := make([]openai.ChatCompletionMessageParamUnion, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case "system":
			out = append(out, openai.SystemMessage(m.Content))
		case "user":
			out = append(out, openai.UserMessage(m.Content))
		case "assistant":
			tc := extractToolCalls(m)
			if len(tc) > 0 {
				out = append(out, openai.ChatCompletionMessageParamUnion{
					OfAssistant: &openai.ChatCompletionAssistantMessageParam{
						Content: openai.ChatCompletionAssistantMessageParamContentUnion{
							OfString: openai.String(m.Content),
						},
						ToolCalls: tc,
					},
				})
			} else {
				out = append(out, openai.AssistantMessage(m.Content))
			}
		case "tool":
			id, content := "", m.Content
			for _, p := range m.Parts {
				if p.Type == "tool_result" {
					id = p.ToolCallID
					content = p.Result
					break
				}
			}
			out = append(out, openai.ToolMessage(id, content))
		}
	}
	return out
}

func extractToolCalls(m Message) []openai.ChatCompletionMessageToolCallParam {
	var out []openai.ChatCompletionMessageToolCallParam
	for _, p := range m.Parts {
		if p.Type != "tool_call" {
			continue
		}
		raw, _ := json.Marshal(p.Args)
		out = append(out, openai.ChatCompletionMessageToolCallParam{
			ID:   p.ToolCallID,
			Type: "function",
			Function: openai.ChatCompletionMessageToolCallFunctionParam{
				Name:      p.ToolName,
				Arguments: string(raw),
			},
		})
	}
	return out
}

func (o *OpenAI) mapResponse(comp *openai.ChatCompletion) *Response {
	if len(comp.Choices) == 0 {
		return &Response{Model: string(comp.Model), FinishReason: "stop", Message: Message{Role: "assistant"}}
	}
	r := o.mapChoice(comp.Choices[0], string(comp.Model))
	r.Usage = Usage{
		PromptTokens:     int(comp.Usage.PromptTokens),
		CompletionTokens: int(comp.Usage.CompletionTokens),
	}
	return r
}

func (o *OpenAI) mapChoice(ch openai.ChatCompletionChoice, model string) *Response {
	msg := Message{Role: "assistant", Content: ch.Message.Content}
	finish := mapFinishReason(ch.FinishReason)
	for _, tc := range ch.Message.ToolCalls {
		var args map[string]any
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		msg.Parts = append(msg.Parts, Part{
			Type:       "tool_call",
			ToolCallID: tc.ID,
			ToolName:   tc.Function.Name,
			Args:       args,
		})
	}
	return &Response{Message: msg, Model: model, FinishReason: finish}
}

func mapFinishReason(r string) string {
	switch r {
	case "tool_calls":
		return "tool_calls"
	case "length":
		return "length"
	default:
		return "stop"
	}
}

var _ Provider = (*OpenAI)(nil)
