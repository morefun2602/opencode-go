package tool

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/morefun2602/opencode-go/internal/tools"
)

type fakeTaskRunner struct {
	lastAgent string
}

func (f *fakeTaskRunner) CompleteTurn(ctx context.Context, workspaceID, sessionID, userText string) (string, error) {
	if agent, ok := SubagentNameFromContext(ctx); ok {
		f.lastAgent = agent
	}
	return "ok", nil
}

func (f *fakeTaskRunner) CreateSession(ctx context.Context, workspaceID string) (string, error) {
	return "task-session", nil
}

type fakeLookup struct{}

func (fakeLookup) SessionExists(ctx context.Context, workspaceID, sessionID string) (bool, error) {
	return sessionID == "task-session", nil
}

func TestTaskToolSetsSubagentContext(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	reg := tools.New(log)
	runner := &fakeTaskRunner{}
	RegisterTask(
		reg,
		runner,
		fakeLookup{},
		"ws",
		2,
		func() []SubagentInfo {
			return []SubagentInfo{
				{Name: "general", CanUse: true},
				{Name: "explore", CanUse: true},
			}
		},
		func(name string) (SubagentInfo, error) {
			return SubagentInfo{Name: name, CanUse: true}, nil
		},
	)

	out, err := reg.Run(context.Background(), "c1", "s1", "task", map[string]any{
		"prompt":        "do",
		"subagent_type": "explore",
	})
	if err != nil {
		t.Fatalf("task run failed: %v", err)
	}
	if runner.lastAgent != "explore" {
		t.Fatalf("expected subagent explore, got %q", runner.lastAgent)
	}
	if !strings.Contains(out, "task_id: task-session") {
		t.Fatalf("unexpected task output: %q", out)
	}
}
