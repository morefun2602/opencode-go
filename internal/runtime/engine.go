package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/morefun2602/opencode-go/internal/bus"
	"github.com/morefun2602/opencode-go/internal/config"
	"github.com/morefun2602/opencode-go/internal/llm"
	"github.com/morefun2602/opencode-go/internal/permission"
	"github.com/morefun2602/opencode-go/internal/policy"
	"github.com/morefun2602/opencode-go/internal/prompt"
	"github.com/morefun2602/opencode-go/internal/skill"
	"github.com/morefun2602/opencode-go/internal/store"
	"github.com/morefun2602/opencode-go/internal/tool"
	"github.com/morefun2602/opencode-go/internal/tools"
)

const maxStepsWarning = `You are about to reach the maximum number of tool call steps allowed. Please wrap up your current work:
1. Summarize what you have accomplished so far
2. List any remaining tasks
3. Provide your final response

Do NOT make any more tool calls.`

var noopToolDef = llm.ToolDef{
	Name:        "_noop",
	Description: "A no-operation placeholder tool. Call this if no other tool is appropriate.",
	Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
}

type Engine struct {
	Store                store.Store
	LLM                  llm.Provider
	Router               *llm.Router
	Providers            *llm.Registry
	Tools                *tool.Router
	Policy               *policy.Policy
	Log                  *slog.Logger
	Bus                  *bus.Bus
	Skills               []skill.Skill
	Agent                Agent
	AgentSwitch          *AgentSwitch
	Mode                 Mode
	MaxToolRounds        int
	LLMMaxRetries        int
	CompactionTurns      int
	SystemPrompt         string
	WorkspaceRoot        string
	ConfigInstructions   []string
	CompactionConfig     config.CompactionConfig
	Confirm              func(name string, args map[string]any) (bool, error)
	DoomLoopWindow       int
	Snapshot             SnapshotService
	Compaction           CompactionService
	StructuredOutputSchema map[string]any

	sessions sync.Map // sessionID → context.CancelFunc
}

// SnapshotService is an optional interface for workspace snapshots.
type SnapshotService interface {
	Track(ctx context.Context, sessionID, stepID string) error
	Patch(ctx context.Context, sessionID, stepID string) error
}

// CompactionService compresses message history when context overflows.
type CompactionService interface {
	Process(ctx context.Context, provider llm.Provider, workspaceID, sessionID string, msgs []llm.Message, keepRecent int) ([]llm.Message, error)
}

// toolCallSig represents a tool call signature for doom loop detection.
type toolCallSig struct {
	name     string
	argsHash string
}

func computeToolCallSig(name string, args map[string]any) toolCallSig {
	h := sha256.New()
	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(h, "%s=%v;", k, args[k])
	}
	return toolCallSig{name: name, argsHash: hex.EncodeToString(h.Sum(nil)[:8])}
}

func isDoomLoop(window []toolCallSig, size int) bool {
	if len(window) < size {
		return false
	}
	recent := window[len(window)-size:]
	first := recent[0]
	for _, s := range recent[1:] {
		if s.name != first.name || s.argsHash != first.argsHash {
			return false
		}
	}
	return true
}

func (e *Engine) CreateSession(ctx context.Context, workspaceID string) (string, error) {
	return e.Store.CreateSession(ctx, workspaceID)
}

// CancelSession cancels a running session by calling its cancel func.
func (e *Engine) CancelSession(sessionID string) {
	if v, ok := e.sessions.LoadAndDelete(sessionID); ok {
		if cancel, ok := v.(context.CancelFunc); ok {
			cancel()
		}
	}
	if e.Bus != nil {
		e.Bus.Publish("session.abort", map[string]any{"session_id": sessionID})
	}
}

func (e *Engine) CompleteTurn(ctx context.Context, workspaceID, sessionID, userText string) (string, error) {
	ctx, cancel := context.WithCancel(ctx)
	e.sessions.Store(sessionID, cancel)
	defer func() {
		cancel()
		e.sessions.Delete(sessionID)
		if e.AgentSwitch != nil {
			e.AgentSwitch.Delete(sessionID)
		}
	}()

	if e.Log != nil {
		e.Log.Info("turn_start", "session_id", sessionID, "workspace_id", workspaceID)
	}

	msgs, err := e.loadHistory(ctx, workspaceID, sessionID)
	if err != nil {
		return "", err
	}

	agent := e.activeAgent(ctx, sessionID)
	prov, _ := e.resolveProviderForAgent(agent)
	sys := e.buildSystemPromptForAgent(prov, agent)
	if sys != "" {
		msgs = append([]llm.Message{{Role: "system", Content: sys}}, msgs...)
	}
	msgs = append(msgs, llm.Message{Role: "user", Content: userText})

	tdefs := e.collectToolsForAgent(agent)
	tdefs = e.maybeInjectNoop(tdefs, msgs)
	var newMsgs []store.MessageRow
	newMsgs = append(newMsgs, msgToRow(llm.Message{Role: "user", Content: userText}, nil))

	maxR := e.maxRounds(agent)

	doomWindow := e.DoomLoopWindow
	if doomWindow <= 0 {
		doomWindow = 3
	}
	var sigWindow []toolCallSig

	for round := 0; round < maxR; round++ {
		if ctx.Err() != nil {
			break
		}

		if e.Log != nil {
			e.Log.Info("react_round", "round", round+1, "session_id", sessionID)
		}

		stepID := fmt.Sprintf("step-%d", round)
		if e.Snapshot != nil {
			_ = e.Snapshot.Track(ctx, sessionID, stepID)
		}

		agent = e.activeAgent(ctx, sessionID)
		tdefs = e.collectToolsForAgent(agent)
		tdefs = e.maybeInjectNoop(tdefs, msgs)

		roundTdefs := tdefs
		if round == maxR-1 {
			msgs = append(msgs, llm.Message{Role: "user", Content: maxStepsWarning})
			roundTdefs = nil
		}

		resp, err := e.callWithRetry(ctx, sessionID, msgs, roundTdefs)
		if err != nil {
			if llm.Classify(err) == llm.ContextOverflow && e.Compaction != nil {
				if e.Log != nil {
					e.Log.Warn("context_overflow, compacting", "session_id", sessionID)
				}
				compacted, cErr := e.Compaction.Process(ctx, prov, workspaceID, sessionID, msgs, 5)
				if cErr != nil {
					return "", fmt.Errorf("compaction failed: %w", cErr)
				}
				msgs = compacted
				resp, err = e.callWithRetry(ctx, sessionID, msgs, roundTdefs)
				if err != nil {
					return "", err
				}
			} else {
				if e.Log != nil {
					e.Log.Error("turn_fail", "session_id", sessionID, "err", err)
				}
				return "", err
			}
		}

		if e.CompactionConfig.AutoEnabled() && e.Compaction != nil {
			if tools.IsOverflow(resp.Usage, 128000, e.CompactionConfig.ReservedTokens()) {
				if e.CompactionConfig.PruneEnabled() {
					msgs = tools.Prune(msgs, 40000)
				}
				compacted, cErr := e.Compaction.Process(ctx, prov, workspaceID, sessionID, msgs, 5)
				if cErr == nil {
					msgs = compacted
				}
			}
		}

		msgs = append(msgs, resp.Message)
		newMsgs = append(newMsgs, msgToRow(resp.Message, &resp.Usage))

		if resp.FinishReason != "tool_calls" {
			if e.Snapshot != nil {
				_ = e.Snapshot.Patch(ctx, sessionID, stepID)
			}
			break
		}

		for _, p := range resp.Message.Parts {
			if p.Type != "tool_call" {
				continue
			}

			if p.ToolName == "_noop" {
				toolMsg := llm.Message{
					Role: "tool",
					Parts: []llm.Part{{
						Type: "tool_result", ToolCallID: p.ToolCallID, ToolName: p.ToolName, Result: "noop",
					}},
				}
				msgs = append(msgs, toolMsg)
				newMsgs = append(newMsgs, store.MessageRow{
					Role: "tool", Body: "noop", Parts: mustJSON(toolMsg.Parts), ToolCallID: p.ToolCallID,
				})
				continue
			}

			sig := computeToolCallSig(p.ToolName, p.Args)
			sigWindow = append(sigWindow, sig)
			if isDoomLoop(sigWindow, doomWindow) {
				if e.Log != nil {
					e.Log.Warn("doom_loop_detected", "tool", p.ToolName, "session_id", sessionID)
				}
				if e.Confirm != nil {
					ok, _ := e.Confirm("__doom_loop__", map[string]any{
						"tool":  p.ToolName,
						"count": doomWindow,
					})
					if !ok {
						toolMsg := llm.Message{
							Role: "tool",
							Parts: []llm.Part{{
								Type:       "tool_result",
								ToolCallID: p.ToolCallID,
								ToolName:   p.ToolName,
								Result:     "doom loop detected: same tool call repeated, user chose to stop",
								IsError:    true,
							}},
						}
						msgs = append(msgs, toolMsg)
						newMsgs = append(newMsgs, store.MessageRow{
							Role: "tool", Body: toolMsg.Parts[0].Result,
							Parts: mustJSON(toolMsg.Parts), ToolCallID: p.ToolCallID,
						})
						goto persist
					}
					sigWindow = sigWindow[:0]
				}
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

		if e.Snapshot != nil {
			_ = e.Snapshot.Patch(ctx, sessionID, stepID)
		}
	}

persist:
	if len(e.StructuredOutputSchema) > 0 && ctx.Err() == nil {
		soResult, soMsgs := e.requestStructuredOutput(ctx, sessionID, msgs)
		if soResult != "" {
			newMsgs = append(newMsgs, soMsgs...)
			msgs = append(msgs, soMsgs2LLMMessages(soMsgs)...)
		}
	}

	if err := e.Store.AppendMessages(ctx, workspaceID, sessionID, newMsgs); err != nil {
		return "", err
	}
	if e.Bus != nil {
		e.Bus.Publish("message.created", map[string]any{"session_id": sessionID})
	}

	e.maybeCompact(ctx, prov, workspaceID, sessionID)

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
	ctx, cancel := context.WithCancel(ctx)
	e.sessions.Store(sessionID, cancel)
	defer func() {
		cancel()
		e.sessions.Delete(sessionID)
		if e.AgentSwitch != nil {
			e.AgentSwitch.Delete(sessionID)
		}
	}()

	if e.Log != nil {
		e.Log.Info("turn_start", "session_id", sessionID, "workspace_id", workspaceID)
	}

	msgs, err := e.loadHistory(ctx, workspaceID, sessionID)
	if err != nil {
		return err
	}

	agent := e.activeAgent(ctx, sessionID)
	prov, _ := e.resolveProviderForAgent(agent)
	sys := e.buildSystemPromptForAgent(prov, agent)
	if sys != "" {
		msgs = append([]llm.Message{{Role: "system", Content: sys}}, msgs...)
	}
	msgs = append(msgs, llm.Message{Role: "user", Content: userText})

	tdefs := e.collectToolsForAgent(agent)
	tdefs = e.maybeInjectNoop(tdefs, msgs)
	var newMsgs []store.MessageRow
	newMsgs = append(newMsgs, msgToRow(llm.Message{Role: "user", Content: userText}, nil))

	maxR := e.maxRounds(agent)

	doomWindow := e.DoomLoopWindow
	if doomWindow <= 0 {
		doomWindow = 3
	}
	var sigWindow []toolCallSig

	for round := 0; round < maxR; round++ {
		if ctx.Err() != nil {
			break
		}

		agent = e.activeAgent(ctx, sessionID)
		tdefs = e.collectToolsForAgent(agent)
		tdefs = e.maybeInjectNoop(tdefs, msgs)

		stepID := fmt.Sprintf("step-%d", round)
		if e.Snapshot != nil {
			_ = e.Snapshot.Track(ctx, sessionID, stepID)
		}

		roundTdefs := tdefs
		if round == maxR-1 {
			msgs = append(msgs, llm.Message{Role: "user", Content: maxStepsWarning})
			roundTdefs = nil
		}

		resp, err := e.streamWithRetry(ctx, sessionID, msgs, roundTdefs, chunk)
		if err != nil {
			if llm.Classify(err) == llm.ContextOverflow && e.Compaction != nil {
				compacted, cErr := e.Compaction.Process(ctx, prov, workspaceID, sessionID, msgs, 5)
				if cErr != nil {
					return fmt.Errorf("compaction failed: %w", cErr)
				}
				msgs = compacted
				resp, err = e.streamWithRetry(ctx, sessionID, msgs, roundTdefs, chunk)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}

		msgs = append(msgs, resp.Message)
		newMsgs = append(newMsgs, msgToRow(resp.Message, &resp.Usage))

		if resp.FinishReason != "tool_calls" {
			if e.Snapshot != nil {
				_ = e.Snapshot.Patch(ctx, sessionID, stepID)
			}
			break
		}

		for _, p := range resp.Message.Parts {
			if p.Type != "tool_call" {
				continue
			}

			if p.ToolName == "_noop" {
				toolMsg := llm.Message{
					Role: "tool",
					Parts: []llm.Part{{
						Type: "tool_result", ToolCallID: p.ToolCallID, ToolName: p.ToolName, Result: "noop",
					}},
				}
				msgs = append(msgs, toolMsg)
				newMsgs = append(newMsgs, store.MessageRow{
					Role: "tool", Body: "noop", Parts: mustJSON(toolMsg.Parts), ToolCallID: p.ToolCallID,
				})
				continue
			}

			sig := computeToolCallSig(p.ToolName, p.Args)
			sigWindow = append(sigWindow, sig)
			if isDoomLoop(sigWindow, doomWindow) {
				if e.Confirm != nil {
					ok, _ := e.Confirm("__doom_loop__", map[string]any{
						"tool":  p.ToolName,
						"count": doomWindow,
					})
					if !ok {
						toolMsg := llm.Message{
							Role: "tool",
							Parts: []llm.Part{{
								Type:       "tool_result",
								ToolCallID: p.ToolCallID,
								ToolName:   p.ToolName,
								Result:     "doom loop detected: same tool call repeated, user chose to stop",
								IsError:    true,
							}},
						}
						msgs = append(msgs, toolMsg)
						newMsgs = append(newMsgs, store.MessageRow{
							Role: "tool", Body: toolMsg.Parts[0].Result,
							Parts: mustJSON(toolMsg.Parts), ToolCallID: p.ToolCallID,
						})
						goto persistStream
					}
					sigWindow = sigWindow[:0]
				}
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

		if e.Snapshot != nil {
			_ = e.Snapshot.Patch(ctx, sessionID, stepID)
		}
	}

persistStream:
	if len(e.StructuredOutputSchema) > 0 && ctx.Err() == nil {
		soResult, soMsgs := e.requestStructuredOutput(ctx, sessionID, msgs)
		if soResult != "" {
			newMsgs = append(newMsgs, soMsgs...)
			_ = soResult
		}
	}

	if err := e.Store.AppendMessages(ctx, workspaceID, sessionID, newMsgs); err != nil {
		return err
	}
	if e.Bus != nil {
		e.Bus.Publish("message.created", map[string]any{"session_id": sessionID})
	}
	e.maybeCompact(ctx, prov, workspaceID, sessionID)
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

	agent := e.activeAgent(ctx, sessionID)
	if len(agent.Permission) > 0 {
		action := permission.Evaluate(permission.ToolPermissionName(p.ToolName), "*", agent.Permission)
		if action == permission.ActionDeny {
			return fmt.Sprintf("tool %q is denied by agent %q permissions", p.ToolName, agent.Name), true
		}
		if action == permission.ActionAsk {
			if e.Confirm != nil {
				ok, err := e.Confirm(p.ToolName, p.Args)
				if err != nil || !ok {
					return fmt.Sprintf("tool %q rejected by user (agent %q requires approval)", p.ToolName, agent.Name), true
				}
			} else {
				return fmt.Sprintf("tool %q requires approval but no confirm handler (agent %q)", p.ToolName, agent.Name), true
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

func (e *Engine) callWithRetry(ctx context.Context, sessionID string, msgs []llm.Message, tdefs []llm.ToolDef) (*llm.Response, error) {
	prov, _ := e.resolveProvider()
	attempts := e.LLMMaxRetries + 1
	if attempts < 1 {
		attempts = 1
	}
	var last error
	for i := 0; i < attempts; i++ {
		resp, err := prov.Chat(ctx, msgs, tdefs)
		if err == nil {
			return resp, nil
		}
		last = err
		k, wrapped := llm.ClassifyWithRetry(err)
		if k != llm.Timeout && k != llm.RateLimit {
			return nil, err
		}
		delay := llm.RetryDelay(i, wrapped)
		if e.Log != nil {
			e.Log.Warn("llm_retry", "attempt", i+1, "kind", k, "delay_ms", delay.Milliseconds(), "err", err)
		}
		if e.Bus != nil {
			e.Bus.Publish("session.retry", map[string]any{
				"session_id": sessionID,
				"attempt":    i + 1,
				"delay_ms":   delay.Milliseconds(),
				"error":      err.Error(),
			})
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}
	return nil, last
}

func (e *Engine) streamWithRetry(ctx context.Context, sessionID string, msgs []llm.Message, tdefs []llm.ToolDef, chunk func(string) error) (*llm.Response, error) {
	prov, _ := e.resolveProvider()
	attempts := e.LLMMaxRetries + 1
	if attempts < 1 {
		attempts = 1
	}
	streamCb := func(partial *llm.Response) error {
		if partial.Message.Content != "" {
			return chunk(partial.Message.Content)
		}
		return nil
	}
	var last error
	for i := 0; i < attempts; i++ {
		resp, err := prov.ChatStream(ctx, msgs, tdefs, streamCb)
		if err == nil {
			return resp, nil
		}
		last = err
		k, wrapped := llm.ClassifyWithRetry(err)
		if k != llm.Timeout && k != llm.RateLimit {
			return nil, err
		}
		delay := llm.RetryDelay(i, wrapped)
		if e.Log != nil {
			e.Log.Warn("stream_retry", "attempt", i+1, "kind", k, "delay_ms", delay.Milliseconds(), "err", err)
		}
		if e.Bus != nil {
			e.Bus.Publish("session.retry", map[string]any{
				"session_id": sessionID,
				"attempt":    i + 1,
				"delay_ms":   delay.Milliseconds(),
				"error":      err.Error(),
			})
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
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
	return filterCompacted(out), nil
}

// filterCompacted drops messages before the last compaction summary.
func filterCompacted(msgs []llm.Message) []llm.Message {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" && strings.Contains(msgs[i].Content, "[Conversation Summary]") {
			return msgs[i:]
		}
	}
	return msgs
}

// activeAgent returns the currently effective Agent for the given session.
// Priority: context override (sub-agent) > AgentSwitch (plan_enter) > Engine.Agent > AgentBuild.
func (e *Engine) activeAgent(ctx context.Context, sessionID string) Agent {
	if name, ok := tool.SubagentNameFromContext(ctx); ok {
		if a, found := GetAgent(name); found {
			return a
		}
	}
	if e.AgentSwitch != nil {
		if a, ok := e.AgentSwitch.Get(sessionID); ok {
			return a
		}
	}
	if e.Agent.Name != "" {
		return e.Agent
	}
	return AgentBuild
}

func (e *Engine) collectToolsForAgent(agent Agent) []llm.ToolDef {
	var all []llm.ToolDef
	if e.Tools != nil && e.Tools.Builtin != nil {
		for _, t := range e.Tools.Builtin.List() {
			if t.Name == "invalid" {
				continue
			}
			td := llm.ToolDef{Name: t.Name, Description: t.Description, Parameters: t.Schema}
			if len(t.Tags) > 0 {
				if td.Parameters == nil {
					td.Parameters = map[string]any{}
				}
				td.Parameters["_tags"] = t.Tags
			}
			all = append(all, td)
		}
	}
	if e.Tools != nil {
		for _, c := range e.Tools.Clients {
			for _, t := range c.ListTools() {
				all = append(all, llm.ToolDef{Name: t.Name, Description: t.Description, Parameters: t.Schema})
			}
		}
	}

	filtered := ToolFilter(agent, all)

	for i := range filtered {
		delete(filtered[i].Parameters, "_tags")
	}

	return filtered
}

func (e *Engine) resolveProviderForAgent(agent Agent) (llm.Provider, string) {
	if agent.Model != "" && e.Providers != nil {
		parts := strings.SplitN(agent.Model, "/", 2)
		if len(parts) == 2 {
			if prov, err := e.Providers.Get(parts[0]); err == nil {
				return prov, parts[1]
			}
		}
	}
	return e.resolveProvider()
}

func (e *Engine) maxRounds(agent Agent) int {
	if agent.Steps > 0 {
		return agent.Steps
	}
	if e.MaxToolRounds > 0 {
		return e.MaxToolRounds
	}
	return 25
}

func (e *Engine) buildSystemPromptForAgent(prov llm.Provider, agent Agent) string {
	provType := ""
	if prov != nil {
		provType = prov.Name()
	}
	agentPrompt := agent.Prompt

	sys := prompt.Build(prompt.BuildOpts{
		ProviderType:       provType,
		AgentPrompt:        agentPrompt,
		WorkspaceRoot:      e.WorkspaceRoot,
		ConfigInstructions: e.ConfigInstructions,
		Skills:             e.Skills,
	})
	return sys
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

func (e *Engine) resolveProvider() (llm.Provider, string) {
	if e.Router != nil {
		prov, model, err := e.Router.ResolveDefault()
		if err == nil {
			return prov, model
		}
	}
	if e.LLM != nil {
		return e.LLM, ""
	}
	return llm.Stub{}, ""
}

func (e *Engine) resolveSmallProvider() (llm.Provider, string) {
	if e.Router != nil {
		prov, model, err := e.Router.ResolveSmall()
		if err == nil {
			return prov, model
		}
	}
	return e.resolveProvider()
}

func (e *Engine) maybeCompact(ctx context.Context, prov llm.Provider, workspaceID, sessionID string) {
	if e.CompactionTurns <= 0 || e.Compaction == nil {
		return
	}
	rows, err := e.Store.ListMessages(ctx, workspaceID, sessionID, 0, 100000)
	if err != nil {
		if e.Log != nil {
			e.Log.Error("maybeCompact_load_fail", "session_id", sessionID, "err", err)
		}
		return
	}
	if len(rows) <= e.CompactionTurns*2 {
		return
	}
	if e.Log != nil {
		e.Log.Info("compaction_threshold_exceeded", "session_id", sessionID, "messages", len(rows))
	}

	msgs := make([]llm.Message, 0, len(rows))
	for _, r := range rows {
		msg := llm.Message{Role: r.Role, Content: r.Body}
		if r.Parts != "" && r.Parts != "[]" {
			var parts []llm.Part
			if err := json.Unmarshal([]byte(r.Parts), &parts); err == nil {
				msg.Parts = parts
			}
		}
		msgs = append(msgs, msg)
	}

	if prov == nil {
		prov, _ = e.resolveProvider()
	}

	go func() {
		bgCtx := context.Background()
		compacted, cErr := e.Compaction.Process(bgCtx, prov, workspaceID, sessionID, msgs, 5)
		if cErr != nil {
			if e.Log != nil {
				e.Log.Error("maybeCompact_fail", "session_id", sessionID, "err", cErr)
			}
			return
		}
		var compactedRows []store.MessageRow
		for _, m := range compacted {
			compactedRows = append(compactedRows, msgToRow(m, nil))
		}
		if err := e.Store.AppendMessages(bgCtx, workspaceID, sessionID, compactedRows); err != nil {
			if e.Log != nil {
				e.Log.Error("maybeCompact_persist_fail", "session_id", sessionID, "err", err)
			}
		}
	}()
}

// maybeInjectNoop injects a _noop tool definition when tool list is empty
// but message history contains tool_call parts (prevents LLM errors).
func (e *Engine) maybeInjectNoop(tdefs []llm.ToolDef, msgs []llm.Message) []llm.ToolDef {
	if len(tdefs) > 0 {
		return tdefs
	}
	for _, m := range msgs {
		for _, p := range m.Parts {
			if p.Type == "tool_call" {
				return []llm.ToolDef{noopToolDef}
			}
		}
	}
	return tdefs
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

func (e *Engine) requestStructuredOutput(ctx context.Context, sessionID string, msgs []llm.Message) (string, []store.MessageRow) {
	soTool := llm.ToolDef{
		Name:        "_structured_output",
		Description: "Provide your final answer as structured JSON matching the required schema.",
		Parameters:  e.StructuredOutputSchema,
	}

	soPromptMsg := llm.Message{
		Role:    "user",
		Content: "Please provide your final answer using the _structured_output tool with the required JSON schema.",
	}
	soMsgs := append(msgs, soPromptMsg)

	resp, err := e.callWithRetry(ctx, sessionID, soMsgs, []llm.ToolDef{soTool})
	if err != nil {
		if e.Log != nil {
			e.Log.Error("structured_output_fail", "session_id", sessionID, "err", err)
		}
		return "", nil
	}

	var rows []store.MessageRow
	rows = append(rows, msgToRow(resp.Message, &resp.Usage))

	for _, p := range resp.Message.Parts {
		if p.Type == "tool_call" && p.ToolName == "_structured_output" {
			result, _ := json.Marshal(p.Args)
			toolMsg := llm.Message{
				Role: "tool",
				Parts: []llm.Part{{
					Type: "tool_result", ToolCallID: p.ToolCallID, ToolName: p.ToolName, Result: string(result),
				}},
			}
			rows = append(rows, msgToRow(toolMsg, nil))
			return string(result), rows
		}
	}

	return "", rows
}

func soMsgs2LLMMessages(rows []store.MessageRow) []llm.Message {
	var out []llm.Message
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
	return out
}

func (e *Engine) maybeAutoTitle(ctx context.Context, workspaceID, sessionID, userText string) {
	prov, _ := e.resolveSmallProvider()
	if prov == nil {
		return
	}
	msgs, err := e.Store.ListMessages(ctx, workspaceID, sessionID, 0, 10)
	if err != nil || len(msgs) > 4 {
		return
	}
	go func() {
		titleMsgs := []llm.Message{
			{Role: "system", Content: AgentTitle.Prompt},
			{Role: "user", Content: userText},
		}
		resp, err := prov.Chat(context.Background(), titleMsgs, nil)
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
