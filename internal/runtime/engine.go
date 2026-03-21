package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/morefun2602/opencode-go/internal/bus"
	"github.com/morefun2602/opencode-go/internal/llm"
	"github.com/morefun2602/opencode-go/internal/policy"
	"github.com/morefun2602/opencode-go/internal/skill"
	"github.com/morefun2602/opencode-go/internal/store"
	"github.com/morefun2602/opencode-go/internal/tool"
)

type Engine struct {
	Store           store.Store
	LLM             llm.Provider
	Providers       *llm.Registry
	Tools           *tool.Router
	Policy          *policy.Policy
	Log             *slog.Logger
	Bus             *bus.Bus
	Skills          []skill.Skill
	Mode            Mode
	MaxToolRounds   int
	LLMMaxRetries   int
	CompactionTurns int
	SystemPrompt    string
	Confirm         func(name string, args map[string]any) (bool, error)
}

func (e *Engine) CreateSession(ctx context.Context, workspaceID string) (string, error) {
	return e.Store.CreateSession(ctx, workspaceID)
}

func (e *Engine) CompleteTurn(ctx context.Context, workspaceID, sessionID, userText string) (string, error) {
	if e.Log != nil {
		e.Log.Info("turn_start", "session_id", sessionID, "workspace_id", workspaceID)
	}

	msgs, err := e.loadHistory(ctx, workspaceID, sessionID)
	if err != nil {
		return "", err
	}

	sys := skill.InjectPrompt(e.SystemPrompt, e.Skills)
	if sys != "" {
		msgs = append([]llm.Message{{Role: "system", Content: sys}}, msgs...)
	}
	msgs = append(msgs, llm.Message{Role: "user", Content: userText})

	tools := e.collectTools()
	var newMsgs []store.MessageRow
	newMsgs = append(newMsgs, msgToRow(llm.Message{Role: "user", Content: userText}, nil))

	maxRounds := e.MaxToolRounds
	if maxRounds <= 0 {
		maxRounds = 25
	}

	for round := 0; round < maxRounds; round++ {
		if e.Log != nil {
			e.Log.Info("react_round", "round", round+1, "session_id", sessionID)
		}
		resp, err := e.callWithRetry(ctx, msgs, tools)
		if err != nil {
			if e.Log != nil {
				e.Log.Error("turn_fail", "session_id", sessionID, "err", err)
			}
			return "", err
		}

		msgs = append(msgs, resp.Message)
		newMsgs = append(newMsgs, msgToRow(resp.Message, &resp.Usage))

		if resp.FinishReason != "tool_calls" {
			break
		}

		for _, p := range resp.Message.Parts {
			if p.Type != "tool_call" {
				continue
			}
			result, isErr := e.executeTool(ctx, workspaceID, sessionID, p)
			toolMsg := llm.Message{
				Role: "tool",
				Parts: []llm.Part{{
					Type:       "tool_result",
					ToolCallID: p.ToolCallID,
					ToolName:   p.ToolName,
					Result:     result,
					IsError:    isErr,
				}},
			}
			msgs = append(msgs, toolMsg)
			newMsgs = append(newMsgs, store.MessageRow{
				Role:       "tool",
				Body:       result,
				Parts:      mustJSON(toolMsg.Parts),
				ToolCallID: p.ToolCallID,
			})
		}
	}

	if err := e.Store.AppendMessages(ctx, workspaceID, sessionID, newMsgs); err != nil {
		return "", err
	}
	if e.Bus != nil {
		e.Bus.Publish("message.created", map[string]any{"session_id": sessionID})
	}

	e.maybeCompact(ctx, workspaceID, sessionID)

	if e.Log != nil {
		e.Log.Info("turn_complete", "session_id", sessionID)
	}
	if e.Bus != nil {
		e.Bus.Publish("session.updated", map[string]any{"session_id": sessionID})
	}
	e.maybeAutoTitle(ctx, workspaceID, sessionID, userText)

	last := msgs[len(msgs)-1]
	if last.Role == "assistant" {
		return last.Content, nil
	}
	return "", nil
}

func (e *Engine) CompleteTurnStream(ctx context.Context, workspaceID, sessionID, userText string, chunk func(string) error) error {
	if e.Log != nil {
		e.Log.Info("turn_start", "session_id", sessionID, "workspace_id", workspaceID)
	}

	msgs, err := e.loadHistory(ctx, workspaceID, sessionID)
	if err != nil {
		return err
	}

	sys := skill.InjectPrompt(e.SystemPrompt, e.Skills)
	if sys != "" {
		msgs = append([]llm.Message{{Role: "system", Content: sys}}, msgs...)
	}
	msgs = append(msgs, llm.Message{Role: "user", Content: userText})

	tools := e.collectTools()
	var newMsgs []store.MessageRow
	newMsgs = append(newMsgs, msgToRow(llm.Message{Role: "user", Content: userText}, nil))

	maxRounds := e.MaxToolRounds
	if maxRounds <= 0 {
		maxRounds = 25
	}

	for round := 0; round < maxRounds; round++ {
		resp, err := e.LLM.ChatStream(ctx, msgs, tools, func(partial *llm.Response) error {
			if partial.Message.Content != "" {
				return chunk(partial.Message.Content)
			}
			return nil
		})
		if err != nil {
			return err
		}

		msgs = append(msgs, resp.Message)
		newMsgs = append(newMsgs, msgToRow(resp.Message, &resp.Usage))

		if resp.FinishReason != "tool_calls" {
			break
		}

		for _, p := range resp.Message.Parts {
			if p.Type != "tool_call" {
				continue
			}
			result, isErr := e.executeTool(ctx, workspaceID, sessionID, p)
			toolMsg := llm.Message{
				Role: "tool",
				Parts: []llm.Part{{
					Type:       "tool_result",
					ToolCallID: p.ToolCallID,
					ToolName:   p.ToolName,
					Result:     result,
					IsError:    isErr,
				}},
			}
			msgs = append(msgs, toolMsg)
			newMsgs = append(newMsgs, store.MessageRow{
				Role:       "tool",
				Body:       result,
				Parts:      mustJSON(toolMsg.Parts),
				ToolCallID: p.ToolCallID,
			})
		}
	}

	if err := e.Store.AppendMessages(ctx, workspaceID, sessionID, newMsgs); err != nil {
		return err
	}
	if e.Bus != nil {
		e.Bus.Publish("message.created", map[string]any{"session_id": sessionID})
	}
	e.maybeCompact(ctx, workspaceID, sessionID)
	if e.Log != nil {
		e.Log.Info("turn_complete", "session_id", sessionID)
	}
	if e.Bus != nil {
		e.Bus.Publish("session.updated", map[string]any{"session_id": sessionID})
	}
	e.maybeAutoTitle(ctx, workspaceID, sessionID, userText)
	return nil
}

func (e *Engine) executeTool(ctx context.Context, workspaceID, sessionID string, p llm.Part) (result string, isErr bool) {
	if e.Policy != nil {
		perm := e.Policy.CheckPermission(p.ToolName)
		if perm == "deny" {
			return fmt.Sprintf("tool %q is denied by policy", p.ToolName), true
		}
		if perm == "ask" && e.Confirm != nil {
			ok, err := e.Confirm(p.ToolName, p.Args)
			if err != nil || !ok {
				return fmt.Sprintf("tool %q rejected by user", p.ToolName), true
			}
		}
	}
	if e.Log != nil {
		e.Log.Info("tool_start", "tool", p.ToolName, "session_id", sessionID)
	}
	if e.Bus != nil {
		e.Bus.Publish("tool.start", map[string]any{"tool": p.ToolName, "session_id": sessionID})
	}
	out, err := e.Tools.Run(ctx, "", sessionID, p.ToolName, p.Args)
	if e.Log != nil {
		e.Log.Info("tool_end", "tool", p.ToolName, "session_id", sessionID)
	}
	if e.Bus != nil {
		e.Bus.Publish("tool.end", map[string]any{"tool": p.ToolName, "session_id": sessionID})
	}
	if err != nil {
		return err.Error(), true
	}
	return out, false
}

func (e *Engine) callWithRetry(ctx context.Context, msgs []llm.Message, tools []llm.ToolDef) (*llm.Response, error) {
	attempts := e.LLMMaxRetries + 1
	if attempts < 1 {
		attempts = 1
	}
	var last error
	for i := 0; i < attempts; i++ {
		resp, err := e.LLM.Chat(ctx, msgs, tools)
		if err == nil {
			return resp, nil
		}
		last = err
		k := llm.Classify(err)
		if k != llm.Timeout && k != llm.RateLimit {
			return nil, err
		}
		if e.Log != nil {
			e.Log.Warn("llm_retry", "attempt", i+1, "kind", k, "err", err)
		}
	}
	return nil, last
}

func (e *Engine) loadHistory(ctx context.Context, workspaceID, sessionID string) ([]llm.Message, error) {
	rows, err := e.Store.ListMessages(ctx, workspaceID, sessionID, 0, 100000)
	if err != nil {
		return nil, err
	}
	out := make([]llm.Message, 0, len(rows))
	for _, r := range rows {
		msg := llm.Message{Role: r.Role, Content: r.Body}
		if r.Parts != "" && r.Parts != "[]" {
			var parts []llm.Part
			if err := json.Unmarshal([]byte(r.Parts), &parts); err == nil {
				msg.Parts = parts
			}
		}
		out = append(out, msg)
	}
	return out, nil
}

func (e *Engine) collectTools() []llm.ToolDef {
	var out []llm.ToolDef
	allowed := e.Mode.Tags
	if e.Tools != nil && e.Tools.Builtin != nil {
		for _, t := range e.Tools.Builtin.List() {
			if len(allowed) > 0 && !hasOverlap(t.Tags, allowed) {
				continue
			}
			out = append(out, llm.ToolDef{Name: t.Name, Description: t.Description, Parameters: t.Schema})
		}
	}
	if e.Tools != nil {
		for _, c := range e.Tools.Clients {
			for _, t := range c.ListTools() {
				out = append(out, llm.ToolDef{Name: t.Name, Description: t.Description, Parameters: t.Schema})
			}
		}
	}
	return out
}

func hasOverlap(a, b []string) bool {
	for _, x := range a {
		for _, y := range b {
			if x == y {
				return true
			}
		}
	}
	return false
}

func (e *Engine) maybeCompact(ctx context.Context, workspaceID, sessionID string) {
	if e.Log == nil || e.CompactionTurns <= 0 {
		return
	}
	msgs, err := e.Store.ListMessages(ctx, workspaceID, sessionID, 0, 100000)
	if err != nil {
		return
	}
	if len(msgs) > e.CompactionTurns*2 {
		e.Log.Info("compaction_threshold", "session_id", sessionID, "messages", len(msgs))
	}
}

func msgToRow(m llm.Message, usage *llm.Usage) store.MessageRow {
	row := store.MessageRow{
		Role: m.Role,
		Body: m.Content,
	}
	if len(m.Parts) > 0 {
		row.Parts = mustJSON(m.Parts)
	}
	if usage != nil {
		row.CostPromptTokens = usage.PromptTokens
		row.CostCompletionTokens = usage.CompletionTokens
	}
	return row
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func (e *Engine) maybeAutoTitle(ctx context.Context, workspaceID, sessionID, userText string) {
	if e.LLM == nil {
		return
	}
	msgs, err := e.Store.ListMessages(ctx, workspaceID, sessionID, 0, 10)
	if err != nil || len(msgs) > 4 {
		return
	}
	go func() {
		prompt := []llm.Message{
			{Role: "system", Content: "Generate a concise title (max 6 words) for a conversation that starts with the following message. Return only the title text, nothing else."},
			{Role: "user", Content: userText},
		}
		resp, err := e.LLM.Chat(context.Background(), prompt, nil)
		if err != nil {
			return
		}
		title := resp.Message.Content
		if len(title) > 100 {
			title = title[:100]
		}
		_ = e.Store.SetTitle(context.Background(), workspaceID, sessionID, title)
	}()
}
