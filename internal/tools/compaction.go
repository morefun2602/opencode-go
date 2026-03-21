package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/morefun2602/opencode-go/internal/llm"
)

func IsOverflow(usage llm.Usage, modelLimit, reserved int) bool {
	if modelLimit <= 0 {
		return false
	}
	if reserved <= 0 {
		reserved = 20000
	}
	total := usage.PromptTokens + usage.CompletionTokens
	usable := modelLimit - reserved
	return total >= usable
}

func Prune(msgs []llm.Message, keepRecentTokens int) []llm.Message {
	if keepRecentTokens <= 0 {
		keepRecentTokens = 40000
	}

	totalToolResults := 0
	for _, m := range msgs {
		for _, p := range m.Parts {
			if p.Type == "tool_result" && p.Result != "[pruned]" {
				totalToolResults++
			}
		}
	}
	if totalToolResults <= 2 {
		return msgs
	}

	estimatedTokens := 0
	pruneUpTo := len(msgs)
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		msgTokens := len(m.Content) / 4
		for _, p := range m.Parts {
			if p.Type == "tool_result" {
				msgTokens += len(p.Result) / 4
			}
		}
		estimatedTokens += msgTokens
		if estimatedTokens >= keepRecentTokens {
			pruneUpTo = i
			break
		}
	}

	out := make([]llm.Message, len(msgs))
	for i, m := range msgs {
		if i >= pruneUpTo {
			out[i] = m
			continue
		}
		if m.Role != "tool" {
			out[i] = m
			continue
		}
		nm := llm.Message{Role: m.Role, Content: m.Content}
		if len(m.Parts) > 0 {
			nm.Parts = make([]llm.Part, len(m.Parts))
			for j, p := range m.Parts {
				if p.Type == "tool_result" && p.ToolName != "skill" && p.Result != "[pruned]" {
					np := p
					np.Result = "[pruned]"
					nm.Parts[j] = np
				} else {
					nm.Parts[j] = p
				}
			}
		}
		out[i] = nm
	}
	return out
}

type Compactor struct{}

func NewCompactor() *Compactor {
	return &Compactor{}
}

// Process compresses message history by summarizing older messages and keeping
// the most recent keepRecent messages intact.
func (c *Compactor) Process(
	ctx context.Context,
	provider llm.Provider,
	workspaceID, sessionID string,
	msgs []llm.Message,
	keepRecent int,
) ([]llm.Message, error) {
	if keepRecent <= 0 {
		keepRecent = 5
	}

	var systemMsgs []llm.Message
	var conversationMsgs []llm.Message
	for _, m := range msgs {
		if m.Role == "system" {
			systemMsgs = append(systemMsgs, m)
		} else {
			conversationMsgs = append(conversationMsgs, m)
		}
	}

	if len(conversationMsgs) <= keepRecent {
		return msgs, nil
	}

	toCompress := conversationMsgs[:len(conversationMsgs)-keepRecent]
	toKeep := conversationMsgs[len(conversationMsgs)-keepRecent:]

	summary, err := c.summarize(ctx, provider, toCompress)
	if err != nil {
		return nil, fmt.Errorf("summarize: %w", err)
	}

	summaryMsg := llm.Message{
		Role:    "user",
		Content: fmt.Sprintf("[Conversation Summary]\n%s\n[End Summary - conversation continues below]", summary),
	}

	result := make([]llm.Message, 0, len(systemMsgs)+1+len(toKeep))
	result = append(result, systemMsgs...)
	result = append(result, summaryMsg)
	result = append(result, toKeep...)
	return result, nil
}

func (c *Compactor) summarize(ctx context.Context, provider llm.Provider, msgs []llm.Message) (string, error) {
	var sb strings.Builder
	for _, m := range msgs {
		sb.WriteString(fmt.Sprintf("[%s]: ", m.Role))
		if m.Content != "" {
			sb.WriteString(m.Content)
		}
		for _, p := range m.Parts {
			switch p.Type {
			case "tool_call":
				sb.WriteString(fmt.Sprintf("called tool %s", p.ToolName))
			case "tool_result":
				if len(p.Result) > 200 {
					sb.WriteString(fmt.Sprintf("tool %s result: %s...", p.ToolName, p.Result[:200]))
				} else {
					sb.WriteString(fmt.Sprintf("tool %s result: %s", p.ToolName, p.Result))
				}
			}
		}
		sb.WriteString("\n")
	}

	prompt := []llm.Message{
		{Role: "system", Content: "Summarize the following conversation concisely, preserving key decisions, file changes, and important context. Keep the summary under 500 words."},
		{Role: "user", Content: sb.String()},
	}
	resp, err := provider.Chat(ctx, prompt, nil)
	if err != nil {
		return "", err
	}
	return resp.Message.Content, nil
}
