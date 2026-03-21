package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/morefun2602/opencode-go/internal/llm"
)

// SummaryEntry represents a single step summary.
type SummaryEntry struct {
	StepID      string
	ToolCalls   []string
	FilesChanged []string
	Summary     string
}

// SessionSummary manages incremental session summaries.
type SessionSummary struct {
	entries []SummaryEntry
}

func NewSessionSummary() *SessionSummary {
	return &SessionSummary{}
}

// AddEntry appends a step summary.
func (s *SessionSummary) AddEntry(entry SummaryEntry) {
	s.entries = append(s.entries, entry)
}

// Summarize generates a summary for a step based on tool calls and file diffs.
func (s *SessionSummary) Summarize(
	ctx context.Context,
	provider llm.Provider,
	stepID string,
	toolCalls []string,
	fileDiff string,
) (SummaryEntry, error) {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Step: %s\n", stepID)
	fmt.Fprintf(&sb, "Tool calls: %s\n", strings.Join(toolCalls, ", "))
	if fileDiff != "" {
		fmt.Fprintf(&sb, "File changes:\n%s\n", fileDiff)
	}

	entry := SummaryEntry{
		StepID:    stepID,
		ToolCalls: toolCalls,
	}

	if provider != nil && fileDiff != "" {
		prompt := []llm.Message{
			{Role: "system", Content: "Summarize the following step in one sentence, focusing on what changed and why."},
			{Role: "user", Content: sb.String()},
		}
		resp, err := provider.Chat(ctx, prompt, nil)
		if err == nil {
			entry.Summary = resp.Message.Content
		}
	}

	if entry.Summary == "" {
		entry.Summary = fmt.Sprintf("Step %s: %s", stepID, strings.Join(toolCalls, ", "))
	}

	s.AddEntry(entry)
	return entry, nil
}

// GetAll returns all summary entries.
func (s *SessionSummary) GetAll() []SummaryEntry {
	return append([]SummaryEntry{}, s.entries...)
}

// Format returns a readable summary string.
func (s *SessionSummary) Format() string {
	if len(s.entries) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("## Session Summary\n")
	for _, e := range s.entries {
		fmt.Fprintf(&sb, "- **%s**: %s\n", e.StepID, e.Summary)
	}
	return sb.String()
}
