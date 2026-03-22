package tui

import (
	"encoding/json"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/morefun2602/opencode-go/internal/llm"
	"github.com/morefun2602/opencode-go/internal/store"
)

// RenderBlock is a pre-rendered piece of content for the viewport.
type RenderBlock struct {
	Type    string
	Content string
}

// BuildRenderBlocks converts a message list into pre-rendered content blocks.
func BuildRenderBlocks(msgs []store.MessageRow, theme Theme, width int, renderer *glamour.TermRenderer) []RenderBlock {
	resultMap := buildToolResultMap(msgs)
	var blocks []RenderBlock

	for _, m := range msgs {
		switch m.Role {
		case "user":
			blocks = append(blocks, renderUserBlock(m, theme, width))
		case "assistant":
			blocks = append(blocks, renderAssistantBlocks(m, resultMap, theme, width, renderer)...)
		case "tool":
			// tool messages are rendered inline via tool_call cards; skip standalone display
		}
	}
	return blocks
}

func renderUserBlock(m store.MessageRow, theme Theme, width int) RenderBlock {
	prefix := lipgloss.NewStyle().Foreground(theme.Primary).Bold(true).Render("  You")
	separator := lipgloss.NewStyle().Foreground(theme.Border).Render(strings.Repeat("─", min(width-2, 60)))

	bodyStyle := lipgloss.NewStyle().
		Foreground(theme.Text).
		PaddingLeft(2).
		Width(width - 2)

	body := bodyStyle.Render(m.Body)
	return RenderBlock{Type: "user", Content: separator + "\n" + prefix + "\n" + body + "\n"}
}

func renderAssistantBlocks(m store.MessageRow, results map[string]toolResult, theme Theme, w int, renderer *glamour.TermRenderer) []RenderBlock {
	parts := parseParts(m)
	if len(parts) == 0 {
		// 纯文字回复，直接渲染 Body
		if m.Body == "" {
			return nil
		}
		return []RenderBlock{renderTextBlock(m.Body, theme, w, renderer)}
	}

	var blocks []RenderBlock

	// 始终先渲染文字内容（Body），再追加工具卡片
	if m.Body != "" {
		blocks = append(blocks, renderTextBlock(m.Body, theme, w, renderer))
	}

	for _, p := range parts {
		if p.Type == "tool_call" {
			tr := results[p.ToolCallID]
			card := RenderToolCard(p.ToolName, p.Args, tr.result, tr.isErr, w, theme)
			blocks = append(blocks, RenderBlock{Type: "tool_call", Content: card})
		}
	}

	return blocks
}

func renderTextBlock(text string, theme Theme, w int, renderer *glamour.TermRenderer) RenderBlock {
	prefix := lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true).Render("  Assistant")
	body := text

	if IsDiffContent(body) {
		diffRendered := RenderDiff(body, w-4, theme)
		return RenderBlock{Type: "text", Content: prefix + "\n" + diffRendered + "\n"}
	}

	if renderer != nil {
		if rendered, err := renderer.Render(body); err == nil {
			body = strings.TrimSpace(rendered)
		}
	}

	bodyStyle := lipgloss.NewStyle().PaddingLeft(2).Width(w - 2)
	return RenderBlock{Type: "text", Content: prefix + "\n" + bodyStyle.Render(body) + "\n"}
}

type toolResult struct {
	result string
	isErr  bool
}

func buildToolResultMap(msgs []store.MessageRow) map[string]toolResult {
	m := make(map[string]toolResult)
	for _, msg := range msgs {
		if msg.Role != "tool" {
			continue
		}
		parts := parseParts(msg)
		for _, p := range parts {
			if p.Type == "tool_result" && p.ToolCallID != "" {
				m[p.ToolCallID] = toolResult{result: p.Result, isErr: p.IsError}
			}
		}
		if msg.ToolCallID != "" {
			if _, exists := m[msg.ToolCallID]; !exists {
				m[msg.ToolCallID] = toolResult{result: msg.Body}
			}
		}
	}
	return m
}

func parseParts(m store.MessageRow) []llm.Part {
	if m.Parts == "" || m.Parts == "[]" {
		return nil
	}
	var parts []llm.Part
	if err := json.Unmarshal([]byte(m.Parts), &parts); err != nil {
		return nil
	}
	return parts
}

// BlocksToString joins render blocks into a single string for the viewport.
func BlocksToString(blocks []RenderBlock) string {
	var sb strings.Builder
	for _, b := range blocks {
		sb.WriteString(b.Content)
		sb.WriteString("\n")
	}
	return sb.String()
}
