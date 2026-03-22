package runtime

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	tmpDir, err := os.MkdirTemp("", "react-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	p := filepath.Join(tmpDir, "test.db")
	st, err := store.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	wsDir := filepath.Join(tmpDir, "ws")
	_ = os.MkdirAll(wsDir, 0o755)
	pol := &policy.Policy{WorkspaceRoot: wsDir}
	reg := tools.New(log)
	tool.RegisterBuiltin(reg, pol, nil, nil)

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

func TestFilterCompacted(t *testing.T) {
	msgs := []llm.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
		{Role: "user", Content: "[Conversation Summary] old chat summary"},
		{Role: "assistant", Content: "continuing"},
		{Role: "user", Content: "new question"},
	}
	result := filterCompacted(msgs)
	if len(result) != 3 {
		t.Fatalf("expected 3 messages after filterCompacted, got %d", len(result))
	}
	if result[0].Content != "[Conversation Summary] old chat summary" {
		t.Fatalf("first message should be compaction summary, got %q", result[0].Content)
	}
}

func TestFilterCompactedNoSummary(t *testing.T) {
	msgs := []llm.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}
	result := filterCompacted(msgs)
	if len(result) != 2 {
		t.Fatalf("expected 2 messages (unchanged), got %d", len(result))
	}
}

func TestMaxStepsWarningInjection(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	p := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	callCount := 0
	lastTdefs := []llm.ToolDef{}
	lastMsgs := []llm.Message{}
	trackingProv := &trackingProvider{
		onChat: func(ctx context.Context, msgs []llm.Message, td []llm.ToolDef) (*llm.Response, error) {
			callCount++
			lastTdefs = td
			lastMsgs = msgs
			if callCount < 2 {
				return &llm.Response{
					Message: llm.Message{
						Role: "assistant",
						Parts: []llm.Part{{
							Type: "tool_call", ToolCallID: "tc", ToolName: "echo", Args: map[string]any{},
						}},
					},
					FinishReason: "tool_calls",
					Model:        "track",
				}, nil
			}
			return &llm.Response{
				Message:      llm.Message{Role: "assistant", Content: "final summary"},
				FinishReason: "stop",
				Model:        "track",
			}, nil
		},
	}

	reg := tools.New(log)
	reg.Register(tools.Tool{
		Name: "echo",
		Fn:   func(ctx context.Context, args map[string]any) (string, error) { return "ok", nil },
	})

	eng := &Engine{
		Store:         st,
		LLM:           trackingProv,
		Tools:         &tool.Router{Builtin: reg, Log: log},
		Log:           log,
		MaxToolRounds: 2,
	}

	ctx := context.Background()
	sid, _ := st.CreateSession(ctx, "ws")
	reply, err := eng.CompleteTurn(ctx, "ws", sid, "test")
	if err != nil {
		t.Fatal(err)
	}
	if reply != "final summary" {
		t.Fatalf("expected 'final summary', got %q", reply)
	}

	if len(lastTdefs) != 0 {
		t.Fatalf("expected empty tool defs on last round, got %d", len(lastTdefs))
	}
	found := false
	for _, m := range lastMsgs {
		if m.Role == "user" && m.Content == maxStepsWarning {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected MAX_STEPS warning message in last round messages")
	}
}

type trackingProvider struct {
	onChat func(ctx context.Context, msgs []llm.Message, td []llm.ToolDef) (*llm.Response, error)
}

func (tp *trackingProvider) Name() string    { return "tracking" }
func (tp *trackingProvider) Models() []string { return []string{"tracking"} }

func (tp *trackingProvider) Chat(ctx context.Context, msgs []llm.Message, td []llm.ToolDef) (*llm.Response, error) {
	return tp.onChat(ctx, msgs, td)
}

func (tp *trackingProvider) ChatStream(ctx context.Context, msgs []llm.Message, td []llm.ToolDef, chunk func(*llm.Response) error) (*llm.Response, error) {
	r, err := tp.onChat(ctx, msgs, td)
	if err != nil {
		return nil, err
	}
	_ = chunk(r)
	return r, nil
}

func TestCancelSession(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	p := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	started := make(chan struct{})
	slowProv := &trackingProvider{
		onChat: func(ctx context.Context, msgs []llm.Message, td []llm.ToolDef) (*llm.Response, error) {
			select {
			case started <- struct{}{}:
			default:
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(10 * time.Second):
				return &llm.Response{
					Message:      llm.Message{Role: "assistant", Content: "done"},
					FinishReason: "stop",
				}, nil
			}
		},
	}

	eng := &Engine{
		Store:         st,
		LLM:           slowProv,
		Tools:         &tool.Router{Builtin: tools.New(log), Log: log},
		Log:           log,
		MaxToolRounds: 5,
	}

	ctx := context.Background()
	sid, _ := st.CreateSession(ctx, "ws")

	done := make(chan error, 1)
	go func() {
		_, err := eng.CompleteTurn(ctx, "ws", sid, "test cancel")
		done <- err
	}()

	<-started
	eng.CancelSession(sid)

	select {
	case err := <-done:
		if err == nil || err != context.Canceled {
			t.Logf("got error: %v (expected context.Canceled or similar)", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("CompleteTurn did not terminate after CancelSession")
	}
}

func TestNoopToolInjection(t *testing.T) {
	msgs := []llm.Message{
		{Role: "assistant", Parts: []llm.Part{{Type: "tool_call", ToolName: "read"}}},
		{Role: "tool", Parts: []llm.Part{{Type: "tool_result", ToolName: "read", Result: "content"}}},
	}

	eng := &Engine{}
	result := eng.maybeInjectNoop(nil, msgs)
	if len(result) != 1 || result[0].Name != "_noop" {
		t.Fatalf("expected _noop injection, got %v", result)
	}

	result2 := eng.maybeInjectNoop([]llm.ToolDef{{Name: "read"}}, msgs)
	if len(result2) != 1 || result2[0].Name != "read" {
		t.Fatalf("should not inject noop when tools exist, got %v", result2)
	}
}

func TestNoopToolInjectionNoToolCalls(t *testing.T) {
	msgs := []llm.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}

	eng := &Engine{}
	result := eng.maybeInjectNoop(nil, msgs)
	if len(result) != 0 {
		t.Fatalf("should not inject noop when no tool_call in history, got %v", result)
	}
}

type modelAwareProvider struct {
	name       string
	models     []string
	usedModels []string
}

func (m *modelAwareProvider) Name() string     { return m.name }
func (m *modelAwareProvider) Models() []string { return m.models }
func (m *modelAwareProvider) Chat(ctx context.Context, msgs []llm.Message, td []llm.ToolDef) (*llm.Response, error) {
	// Legacy path should not be used after fix, but keep behavior.
	m.usedModels = append(m.usedModels, "legacy")
	return &llm.Response{
		Message:      llm.Message{Role: "assistant", Content: "ok"},
		FinishReason: "stop",
		Model:        "legacy",
	}, nil
}
func (m *modelAwareProvider) ChatStream(ctx context.Context, msgs []llm.Message, td []llm.ToolDef, chunk func(*llm.Response) error) (*llm.Response, error) {
	resp, _ := m.Chat(ctx, msgs, td)
	_ = chunk(resp)
	return resp, nil
}
func (m *modelAwareProvider) ChatWithModel(ctx context.Context, model string, msgs []llm.Message, td []llm.ToolDef) (*llm.Response, error) {
	m.usedModels = append(m.usedModels, model)
	return &llm.Response{
		Message:      llm.Message{Role: "assistant", Content: "ok"},
		FinishReason: "stop",
		Model:        model,
	}, nil
}
func (m *modelAwareProvider) ChatStreamWithModel(ctx context.Context, model string, msgs []llm.Message, td []llm.ToolDef, chunk func(*llm.Response) error) (*llm.Response, error) {
	m.usedModels = append(m.usedModels, model)
	resp := &llm.Response{
		Message:      llm.Message{Role: "assistant", Content: "ok"},
		FinishReason: "stop",
		Model:        model,
	}
	_ = chunk(resp)
	return resp, nil
}

func TestSetModelAffectsCompleteTurnModelRouting(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	p := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	prov := &modelAwareProvider{name: "opencode", models: []string{"glm-5-free", "gpt-5-nano"}}
	reg := llm.NewRegistry()
	reg.Register("opencode", func() llm.Provider { return prov })
	router := llm.NewRouter(reg, "opencode/glm-5-free", "")

	eng := &Engine{
		Store:         st,
		Router:        router,
		Providers:     reg,
		Log:           log,
		MaxToolRounds: 2,
	}

	eng.SetModel("opencode/gpt-5-nano")

	ctx := context.Background()
	sid, _ := st.CreateSession(ctx, "ws")
	_, err = eng.CompleteTurn(ctx, "ws", sid, "hello")
	if err != nil {
		t.Fatal(err)
	}
	if len(prov.usedModels) == 0 {
		t.Fatal("expected model-aware provider to be called")
	}
	if prov.usedModels[0] != "gpt-5-nano" {
		t.Fatalf("expected routed model gpt-5-nano, got %q", prov.usedModels[0])
	}
}

func TestSetModelAffectsCompleteTurnStreamModelRouting(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	p := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	prov := &modelAwareProvider{name: "opencode", models: []string{"glm-5-free", "gpt-5-nano"}}
	reg := llm.NewRegistry()
	reg.Register("opencode", func() llm.Provider { return prov })
	router := llm.NewRouter(reg, "opencode/glm-5-free", "")

	eng := &Engine{
		Store:         st,
		Router:        router,
		Providers:     reg,
		Log:           log,
		MaxToolRounds: 2,
	}

	eng.SetModel("opencode/gpt-5-nano")

	ctx := context.Background()
	sid, _ := st.CreateSession(ctx, "ws")
	err = eng.CompleteTurnStream(ctx, "ws", sid, "hello", func(string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if len(prov.usedModels) == 0 {
		t.Fatal("expected model-aware provider to be called")
	}
	if prov.usedModels[0] != "gpt-5-nano" {
		t.Fatalf("expected routed model gpt-5-nano, got %q", prov.usedModels[0])
	}
}
