package llm

import (
	"context"
	"strings"
)

// Stub 占位实现，用于开发与测试（不发起外网请求）。
type Stub struct{}

func (Stub) Name() string   { return "stub" }
func (Stub) Models() []string { return []string{"stub"} }

func (Stub) Chat(ctx context.Context, msgs []Message, tools []ToolDef) (*Response, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	text := lastUserText(msgs)
	return &Response{
		Message:      Message{Role: "assistant", Content: "echo: " + text},
		FinishReason: "stop",
		Model:        "stub",
	}, nil
}

func (Stub) ChatStream(ctx context.Context, msgs []Message, tools []ToolDef, chunk func(*Response) error) (*Response, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	text := lastUserText(msgs)
	resp := &Response{
		Message:      Message{Role: "assistant", Content: "echo: " + text},
		FinishReason: "stop",
		Model:        "stub",
	}
	if err := chunk(resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func lastUserText(msgs []Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			if msgs[i].Content != "" {
				return msgs[i].Content
			}
			var sb strings.Builder
			for _, p := range msgs[i].Parts {
				if p.Type == "text" {
					sb.WriteString(p.Text)
				}
			}
			return sb.String()
		}
	}
	return ""
}

var _ Provider = Stub{}
