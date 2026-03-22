package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/morefun2602/opencode-go/internal/bus"
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

func (f *fakeProvider) Name() string     { return "fake" }
func (f *fakeProvider) Models() []string { return []string{"fake"} }

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

func (a *alwaysToolCallProvider) Name() string     { return "always" }
func (a *alwaysToolCallProvider) Models() []string { return []string{"always"} }

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

func (tp *trackingProvider) Name() string     { return "tracking" }
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

func TestInferPermissionArg(t *testing.T) {
	arg := inferPermissionArg(map[string]any{
		"foo":  "bar",
		"path": "a/b/c.txt",
	})
	if arg != "a/b/c.txt" {
		t.Fatalf("expected path arg, got %q", arg)
	}

	arg = inferPermissionArg(map[string]any{
		"cmd": "ls -la",
	})
	if arg != "ls -la" {
		t.Fatalf("expected cmd arg, got %q", arg)
	}
}

func TestExecuteToolPolicyArgPatternDeny(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	reg := tools.New(log)
	reg.Register(tools.Tool{
		Name: "read",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string"},
			},
			"required": []string{"path"},
		},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			return "ok", nil
		},
	})

	eng := &Engine{
		Tools:  &tool.Router{Builtin: reg, Log: log},
		Policy: &policy.Policy{Permissions: map[string]string{"read:*.secret": "deny"}, Log: log},
		Log:    log,
	}
	out, isErr := eng.executeTool(context.Background(), "ws", "sid", llm.Part{
		Type:     "tool_call",
		ToolName: "read",
		Args:     map[string]any{"path": "token.secret"},
	})
	if !isErr {
		t.Fatalf("expected deny error, got output=%q", out)
	}
	if out == "" {
		t.Fatal("expected deny message")
	}
}

func TestApplyToolRoutingStrategyGPT(t *testing.T) {
	in := []llm.ToolDef{
		{Name: "apply_patch"},
		{Name: "edit"},
		{Name: "write"},
		{Name: "read"},
	}
	out := applyToolRoutingStrategy(in, "gpt-5-codex")
	has := map[string]bool{}
	for _, t2 := range out {
		has[t2.Name] = true
	}
	if !has["apply_patch"] {
		t.Fatal("gpt model should keep apply_patch")
	}
	if has["edit"] || has["write"] {
		t.Fatal("gpt model should hide edit/write")
	}
}

func TestApplyToolRoutingStrategyNonGPT(t *testing.T) {
	in := []llm.ToolDef{
		{Name: "apply_patch"},
		{Name: "edit"},
		{Name: "write"},
	}
	out := applyToolRoutingStrategy(in, "claude-sonnet-4")
	has := map[string]bool{}
	for _, t2 := range out {
		has[t2.Name] = true
	}
	if has["apply_patch"] {
		t.Fatal("non-gpt model should hide apply_patch")
	}
	if !has["edit"] || !has["write"] {
		t.Fatal("non-gpt model should keep edit/write")
	}
}

func TestExecuteToolPolicyAskWithoutConfirmRejected(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	reg := tools.New(log)
	reg.Register(tools.Tool{
		Name: "read",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string"},
			},
			"required": []string{"path"},
		},
		Fn: func(ctx context.Context, args map[string]any) (string, error) { return "ok", nil },
	})
	eng := &Engine{
		Tools:  &tool.Router{Builtin: reg, Log: log},
		Policy: &policy.Policy{Permissions: map[string]string{"read": "ask"}, Log: log},
		Log:    log,
	}
	out, isErr := eng.executeTool(context.Background(), "ws", "sid", llm.Part{
		Type:     "tool_call",
		ToolName: "read",
		Args:     map[string]any{"path": "x.txt"},
	})
	if !isErr {
		t.Fatalf("expected ask-without-confirm to be rejected, got %q", out)
	}
	if out == "" {
		t.Fatal("expected rejection reason")
	}
}

func TestDoomLoopWithoutConfirmStopsSession(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	p := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	reg := tools.New(log)
	reg.Register(tools.Tool{
		Name: "echo",
		Fn:   func(ctx context.Context, args map[string]any) (string, error) { return "ok", nil },
	})
	eng := &Engine{
		Store:          st,
		LLM:            &alwaysToolCallProvider{},
		Tools:          &tool.Router{Builtin: reg, Log: log},
		Log:            log,
		MaxToolRounds:  10,
		DoomLoopWindow: 3,
		Confirm:        nil,
	}
	ctx := context.Background()
	sid, _ := st.CreateSession(ctx, "ws")
	_, _ = eng.CompleteTurn(ctx, "ws", sid, "loop")

	rows, _ := st.ListMessages(ctx, "ws", sid, 0, 200)
	found := false
	for _, r := range rows {
		if strings.Contains(r.Body, "doom loop detected: no confirm handler available, stopping") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected doom-loop no-confirm rejection message")
	}
}

func TestDoomLoopWindowRespected(t *testing.T) {
	makeEngine := func(window int, db string) (*Engine, store.Store) {
		log := slog.New(slog.NewTextHandler(io.Discard, nil))
		st, err := store.Open(db)
		if err != nil {
			t.Fatal(err)
		}
		reg := tools.New(log)
		reg.Register(tools.Tool{
			Name: "echo",
			Fn:   func(ctx context.Context, args map[string]any) (string, error) { return "ok", nil },
		})
		eng := &Engine{
			Store:          st,
			LLM:            &alwaysToolCallProvider{},
			Tools:          &tool.Router{Builtin: reg, Log: log},
			Log:            log,
			MaxToolRounds:  10,
			DoomLoopWindow: window,
			Confirm:        nil,
		}
		return eng, st
	}

	countCalls := func(rows []store.MessageRow) int {
		total := 0
		for _, r := range rows {
			if r.Role != "assistant" {
				continue
			}
			var parts []llm.Part
			_ = json.Unmarshal([]byte(r.Parts), &parts)
			for _, p := range parts {
				if p.Type == "tool_call" && p.ToolName == "echo" {
					total++
				}
			}
		}
		return total
	}

	ctx := context.Background()
	eng2, st2 := makeEngine(2, filepath.Join(t.TempDir(), "w2.db"))
	defer st2.Close()
	sid2, _ := st2.CreateSession(ctx, "ws")
	_, _ = eng2.CompleteTurn(ctx, "ws", sid2, "loop")
	rows2, _ := st2.ListMessages(ctx, "ws", sid2, 0, 200)

	eng4, st4 := makeEngine(4, filepath.Join(t.TempDir(), "w4.db"))
	defer st4.Close()
	sid4, _ := st4.CreateSession(ctx, "ws")
	_, _ = eng4.CompleteTurn(ctx, "ws", sid4, "loop")
	rows4, _ := st4.ListMessages(ctx, "ws", sid4, 0, 200)

	if countCalls(rows2) >= countCalls(rows4) {
		t.Fatalf("expected smaller doom window to stop earlier: w2=%d w4=%d", countCalls(rows2), countCalls(rows4))
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

func TestActiveAgentPriority(t *testing.T) {
	eng := &Engine{
		Agent:       AgentBuild,
		AgentSwitch: NewAgentSwitch(),
	}
	eng.AgentSwitch.Set("s1", AgentPlan)

	// session override should apply when no subagent context exists
	a := eng.activeAgent(context.Background(), "s1")
	if a.Name != "plan" {
		t.Fatalf("expected plan agent, got %s", a.Name)
	}

	// subagent context should override AgentSwitch
	ctx := tool.WithSubagentContext(context.Background(), "explore")
	a = eng.activeAgent(ctx, "s1")
	if a.Name != "explore" {
		t.Fatalf("expected explore agent, got %s", a.Name)
	}

	eng.AgentSwitch.Delete("s1")
	a = eng.activeAgent(context.Background(), "s1")
	if a.Name != "build" {
		t.Fatalf("expected build agent fallback, got %s", a.Name)
	}
}

type overflowThenStopProvider struct {
	calls int
}

func (p *overflowThenStopProvider) Name() string     { return "overflow-provider" }
func (p *overflowThenStopProvider) Models() []string { return []string{"m1"} }
func (p *overflowThenStopProvider) Chat(ctx context.Context, msgs []llm.Message, td []llm.ToolDef) (*llm.Response, error) {
	p.calls++
	if p.calls == 1 {
		return nil, errors.New("context window exceeded")
	}
	return &llm.Response{
		Message:      llm.Message{Role: "assistant", Content: "ok"},
		FinishReason: "stop",
		Model:        "m1",
	}, nil
}
func (p *overflowThenStopProvider) ChatStream(ctx context.Context, msgs []llm.Message, td []llm.ToolDef, chunk func(*llm.Response) error) (*llm.Response, error) {
	r, err := p.Chat(ctx, msgs, td)
	if err != nil {
		return nil, err
	}
	_ = chunk(r)
	return r, nil
}

type fakeCompactor struct {
	calls     int
	providers []llm.Provider
}

func (f *fakeCompactor) Process(ctx context.Context, provider llm.Provider, workspaceID, sessionID string, msgs []llm.Message, keepRecent int) ([]llm.Message, error) {
	f.calls++
	f.providers = append(f.providers, provider)
	return msgs, nil
}

func TestContextOverflowTriggersCompactionWithRunProvider(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	prov := &overflowThenStopProvider{}
	comp := &fakeCompactor{}
	b := bus.New()
	ch := b.Subscribe("*")
	defer b.Unsubscribe("*", ch)
	eng := &Engine{
		Store:         st,
		LLM:           prov,
		Tools:         &tool.Router{Builtin: tools.New(log), Log: log},
		Log:           log,
		Compaction:    comp,
		Bus:           b,
		MaxToolRounds: 2,
	}

	ctx := context.Background()
	sid, _ := st.CreateSession(ctx, "ws")
	_, err = eng.CompleteTurn(ctx, "ws", sid, "trigger overflow")
	if err != nil {
		t.Fatal(err)
	}
	if comp.calls != 1 {
		t.Fatalf("expected 1 compaction call, got %d", comp.calls)
	}
	if comp.providers[0] == nil {
		t.Fatal("expected provider to be passed to compaction")
	}
	if prov.calls < 2 {
		t.Fatalf("expected provider to be retried after compaction, calls=%d", prov.calls)
	}
	seenCompact := false
	timeout := time.After(500 * time.Millisecond)
	for !seenCompact {
		select {
		case evt := <-ch:
			if evt.Type == "react.compact.success" {
				seenCompact = true
			}
		case <-timeout:
			t.Fatal("expected react.compact.success event")
		}
	}
}

func TestRuntimeEventsPublished(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	b := bus.New()
	reg := tools.New(log)
	reg.Register(tools.Tool{
		Name: "echo",
		Fn:   func(ctx context.Context, args map[string]any) (string, error) { return "ok", nil },
	})
	eng := &Engine{
		Store:         st,
		LLM:           &alwaysToolCallProvider{},
		Tools:         &tool.Router{Builtin: reg, Log: log},
		Log:           log,
		Bus:           b,
		MaxToolRounds: 1,
	}
	ch := b.Subscribe("*")
	defer b.Unsubscribe("*", ch)
	ctx := context.Background()
	sid, _ := st.CreateSession(ctx, "ws")
	_, _ = eng.CompleteTurn(ctx, "ws", sid, "x")

	seenStart := false
	seenFinish := false
	seenSummary := false
	timeout := time.After(800 * time.Millisecond)
	for !seenStart || !seenFinish || !seenSummary {
		select {
		case evt := <-ch:
			if evt.Type == "react.round.start" {
				seenStart = true
			}
			if evt.Type == "react.round.finish" {
				seenFinish = true
			}
			if evt.Type == "session.summary" {
				seenSummary = true
			}
		case <-timeout:
			t.Fatalf("expected runtime events start/finish/summary, got start=%v finish=%v summary=%v", seenStart, seenFinish, seenSummary)
		}
	}
}

func TestBlockedEventPublishedOnAskWithoutConfirm(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	b := bus.New()
	reg := tools.New(log)
	reg.Register(tools.Tool{
		Name: "read",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string"},
			},
			"required": []string{"path"},
		},
		Fn: func(ctx context.Context, args map[string]any) (string, error) { return "ok", nil },
	})
	eng := &Engine{
		Tools:  &tool.Router{Builtin: reg, Log: log},
		Policy: &policy.Policy{Permissions: map[string]string{"read": "ask"}, Log: log},
		Bus:    b,
		Log:    log,
	}
	ch := b.Subscribe("*")
	defer b.Unsubscribe("*", ch)
	_, _ = eng.executeTool(context.Background(), "ws", "s1", llm.Part{
		Type:     "tool_call",
		ToolName: "read",
		Args:     map[string]any{"path": "x.txt"},
	})
	timeout := time.After(500 * time.Millisecond)
	for {
		select {
		case evt := <-ch:
			if evt.Type == "react.blocked" {
				return
			}
		case <-timeout:
			t.Fatal("expected react.blocked event")
		}
	}
}
