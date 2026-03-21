package runtime

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/morefun2602/opencode-go/internal/llm"
	"github.com/morefun2602/opencode-go/internal/policy"
	"github.com/morefun2602/opencode-go/internal/store"
	"github.com/morefun2602/opencode-go/internal/tool"
	"github.com/morefun2602/opencode-go/internal/tools"
)

type fakeProvider struct {
	rounds int
	cur    int
}

func (f *fakeProvider) Name() string      { return "fake" }
func (f *fakeProvider) Models() []string   { return []string{"fake"} }

func (f *fakeProvider) Chat(ctx context.Context, msgs []llm.Message, td []llm.ToolDef) (*llm.Response, error) {
	f.cur++
	if f.cur < f.rounds {
		return &llm.Response{
			Message: llm.Message{
				Role: "assistant",
				Parts: []llm.Part{{
					Type:       "tool_call",
					ToolCallID: "tc1",
					ToolName:   "read",
					Args:       map[string]any{"path": "nonexistent.txt"},
				}},
			},
			FinishReason: "tool_calls",
			Model:        "fake",
		}, nil
	}
	return &llm.Response{
		Message:      llm.Message{Role: "assistant", Content: "done"},
		FinishReason: "stop",
		Model:        "fake",
	}, nil
}

func (f *fakeProvider) ChatStream(ctx context.Context, msgs []llm.Message, td []llm.ToolDef, chunk func(*llm.Response) error) (*llm.Response, error) {
	r, err := f.Chat(ctx, msgs, td)
	if err != nil {
		return nil, err
	}
	return r, chunk(r)
}

func TestReActLoopWithToolCalls(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	p := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	pol := &policy.Policy{WorkspaceRoot: t.TempDir()}
	reg := tools.New(log)
	tool.RegisterBuiltin(reg, pol)

	eng := &Engine{
		Store:         st,
		LLM:           &fakeProvider{rounds: 2},
		Tools:         &tool.Router{Builtin: reg, Log: log},
		Policy:        pol,
		Log:           log,
		MaxToolRounds: 10,
	}

	ctx := context.Background()
	sid, err := st.CreateSession(ctx, "ws")
	if err != nil {
		t.Fatal(err)
	}

	reply, err := eng.CompleteTurn(ctx, "ws", sid, "do something")
	if err != nil {
		t.Fatal(err)
	}
	if reply != "done" {
		t.Fatalf("want 'done', got %q", reply)
	}

	msgs, err := st.ListMessages(ctx, "ws", sid, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) < 3 {
		t.Fatalf("expected at least 3 messages (user + tool_result + assistant), got %d", len(msgs))
	}
	t.Logf("persisted %d messages", len(msgs))
	for _, m := range msgs {
		t.Logf("  role=%s body=%q parts=%s", m.Role, m.Body, m.Parts)
	}
}

func TestMaxToolRoundsTermination(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	p := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	always := &alwaysToolCallProvider{}
	reg := tools.New(log)
	reg.Register(tools.Tool{
		Name: "echo",
		Fn:   func(ctx context.Context, args map[string]any) (string, error) { return "ok", nil },
	})

	eng := &Engine{
		Store:         st,
		LLM:           always,
		Tools:         &tool.Router{Builtin: reg, Log: log},
		Log:           log,
		MaxToolRounds: 3,
	}

	ctx := context.Background()
	sid, _ := st.CreateSession(ctx, "ws")
	_, _ = eng.CompleteTurn(ctx, "ws", sid, "loop forever")

	msgs, _ := st.ListMessages(ctx, "ws", sid, 0, 100)
	rounds := 0
	for _, m := range msgs {
		if m.Role == "assistant" {
			var parts []llm.Part
			_ = json.Unmarshal([]byte(m.Parts), &parts)
			for _, p := range parts {
				if p.Type == "tool_call" {
					rounds++
				}
			}
		}
	}
	if rounds > 3 {
		t.Fatalf("expected max 3 tool call rounds, got %d", rounds)
	}
}

type alwaysToolCallProvider struct{}

func (a *alwaysToolCallProvider) Name() string      { return "always" }
func (a *alwaysToolCallProvider) Models() []string   { return []string{"always"} }

func (a *alwaysToolCallProvider) Chat(ctx context.Context, msgs []llm.Message, td []llm.ToolDef) (*llm.Response, error) {
	return &llm.Response{
		Message: llm.Message{
			Role: "assistant",
			Parts: []llm.Part{{
				Type: "tool_call", ToolCallID: "tc", ToolName: "echo", Args: map[string]any{},
			}},
		},
		FinishReason: "tool_calls",
		Model:        "always",
	}, nil
}

func (a *alwaysToolCallProvider) ChatStream(ctx context.Context, msgs []llm.Message, td []llm.ToolDef, chunk func(*llm.Response) error) (*llm.Response, error) {
	r, _ := a.Chat(ctx, msgs, td)
	_ = chunk(r)
	return r, nil
}
