package llm

import (
	"regexp"
	"strings"
)

var anthropicIDRegex = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

func TransformMessages(msgs []Message, providerType string) []Message {
	if len(msgs) == 0 {
		return msgs
	}
	switch providerType {
	case "anthropic":
		return transformAnthropic(msgs)
	default:
		return msgs
	}
}

func transformAnthropic(msgs []Message) []Message {
	out := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == "assistant" && m.Content == "" && len(m.Parts) == 0 {
			continue
		}

		nm := Message{
			Role:    m.Role,
			Content: m.Content,
		}
		if len(m.Parts) > 0 {
			nm.Parts = make([]Part, len(m.Parts))
			for i, p := range m.Parts {
				np := p
				if np.ToolCallID != "" {
					np.ToolCallID = anthropicIDRegex.ReplaceAllString(np.ToolCallID, "_")
					if np.ToolCallID == "" || len(np.ToolCallID) < 1 {
						np.ToolCallID = "id_" + strings.Repeat("0", 8)
					}
				}
				nm.Parts[i] = np
			}
		}
		out = append(out, nm)
	}
	return out
}
