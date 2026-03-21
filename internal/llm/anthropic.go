package llm

import (
	"context"
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type Anthropic struct {
	client anthropic.Client
	model  string
}

type AnthropicConfig struct {
	APIKey string
	Model  string
}

func NewAnthropic(cfg AnthropicConfig) *Anthropic {
	var opts []option.RequestOption
	if cfg.APIKey != "" {
		opts = append(opts, option.WithAPIKey(cfg.APIKey))
	}
	model := cfg.Model
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	return &Anthropic{client: anthropic.NewClient(opts...), model: model}
}

func (a *Anthropic) Name() string { return "anthropic" }

func (a *Anthropic) Models() []string {
	return []string{"claude-sonnet-4-20250514", "claude-3-5-sonnet-20241022", "claude-3-5-haiku-20241022"}
}

func (a *Anthropic) Chat(ctx context.Context, msgs []Message, tools []ToolDef) (*Response, error) {
	sys, content := a.splitSystem(msgs)
	params := a.params(sys, content, tools)
	resp, err := a.client.Messages.New(ctx, params)
	if err != nil {
		return nil, err
	}
	return a.mapResponse(resp), nil
}

func (a *Anthropic) ChatStream(ctx context.Context, msgs []Message, tools []ToolDef, chunk func(*Response) error) (*Response, error) {
	sys, content := a.splitSystem(msgs)
	params := a.params(sys, content, tools)
	stream := a.client.Messages.NewStreaming(ctx, params)
	var acc anthropic.Message
	for stream.Next() {
		evt := stream.Current()
		_ = acc.Accumulate(evt)
		partial := &Response{Message: Message{Role: "assistant"}, Model: a.model, FinishReason: "stop"}
		if evt.Delta.Text != "" {
			partial.Message.Content = evt.Delta.Text
		}
		if err := chunk(partial); err != nil {
			return nil, err
		}
	}
	if err := stream.Err(); err != nil {
		return nil, err
	}
	return a.mapResponse(&acc), nil
}

func (a *Anthropic) splitSystem(msgs []Message) (string, []Message) {
	var sys string
	var rest []Message
	for _, m := range msgs {
		if m.Role == "system" {
			if sys != "" {
				sys += "\n"
			}
			sys += m.Content
		} else {
			rest = append(rest, m)
		}
	}
	return sys, rest
}

func (a *Anthropic) params(sys string, msgs []Message, tools []ToolDef) anthropic.MessageNewParams {
	p := anthropic.MessageNewParams{
		Model:     anthropic.Model(a.model),
		MaxTokens: int64(4096),
		Messages:  a.mapMessages(msgs),
	}
	if sys != "" {
		p.System = []anthropic.TextBlockParam{{Text: sys}}
	}
	if len(tools) > 0 {
		td := make([]anthropic.ToolUnionParam, 0, len(tools))
		for _, t := range tools {
			raw, _ := json.Marshal(t.Parameters)
			td = append(td, anthropic.ToolUnionParam{
				OfTool: &anthropic.ToolParam{
					Name:        t.Name,
					Description: anthropic.String(t.Description),
					InputSchema: anthropic.ToolInputSchemaParam{Properties: raw},
				},
			})
		}
		p.Tools = td
	}
	return p
}

func (a *Anthropic) mapMessages(msgs []Message) []anthropic.MessageParam {
	out := make([]anthropic.MessageParam, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case "user":
			out = append(out, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
		case "assistant":
			var blocks []anthropic.ContentBlockParamUnion
			if m.Content != "" {
				blocks = append(blocks, anthropic.NewTextBlock(m.Content))
			}
			for _, p := range m.Parts {
				if p.Type == "tool_call" {
					raw, _ := json.Marshal(p.Args)
					blocks = append(blocks, anthropic.ContentBlockParamUnion{
						OfToolUse: &anthropic.ToolUseBlockParam{
							ID:    p.ToolCallID,
							Name:  p.ToolName,
							Input: json.RawMessage(raw),
						},
					})
				}
			}
			out = append(out, anthropic.MessageParam{Role: "assistant", Content: blocks})
		case "tool":
			for _, p := range m.Parts {
				if p.Type == "tool_result" {
					block := anthropic.ContentBlockParamUnion{
						OfToolResult: &anthropic.ToolResultBlockParam{
							ToolUseID: p.ToolCallID,
							Content: []anthropic.ToolResultBlockParamContentUnion{
								{OfText: &anthropic.TextBlockParam{Text: p.Result}},
							},
							IsError: anthropic.Bool(p.IsError),
						},
					}
					out = append(out, anthropic.NewUserMessage(block))
				}
			}
		}
	}
	return out
}

func (a *Anthropic) mapResponse(resp *anthropic.Message) *Response {
	msg := Message{Role: "assistant"}
	finish := "stop"
	if resp.StopReason == anthropic.StopReasonToolUse {
		finish = "tool_calls"
	} else if resp.StopReason == anthropic.StopReasonMaxTokens {
		finish = "length"
	}
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			if msg.Content != "" {
				msg.Content += "\n"
			}
			msg.Content += block.Text
		case "tool_use":
			var args map[string]any
			_ = json.Unmarshal(block.Input, &args)
			msg.Parts = append(msg.Parts, Part{
				Type:       "tool_call",
				ToolCallID: block.ID,
				ToolName:   block.Name,
				Args:       args,
			})
		}
	}
	return &Response{
		Message:      msg,
		Model:        string(resp.Model),
		FinishReason: finish,
		Usage: Usage{
			PromptTokens:     int(resp.Usage.InputTokens),
			CompletionTokens: int(resp.Usage.OutputTokens),
		},
	}
}

var _ Provider = (*Anthropic)(nil)
