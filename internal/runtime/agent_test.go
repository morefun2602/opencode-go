package runtime

import (
	"testing"

	"github.com/morefun2602/opencode-go/internal/llm"
)

func TestGetAgent(t *testing.T) {
	a, ok := GetAgent("build")
	if !ok {
		t.Fatal("build agent should exist")
	}
	if a.Name != "build" {
		t.Errorf("expected 'build', got %q", a.Name)
	}

	_, ok = GetAgent("nonexistent")
	if ok {
		t.Error("nonexistent agent should not be found")
	}
}

func TestListAgents(t *testing.T) {
	agents := ListAgents()
	if len(agents) == 0 {
		t.Fatal("should have at least one agent")
	}
	for _, a := range agents {
		if a.Hidden {
			t.Errorf("ListAgents should not return hidden agents, got %q", a.Name)
		}
	}
}

func TestToolFilterDenyAll(t *testing.T) {
	tools := []llm.ToolDef{
		{Name: "read"},
		{Name: "write"},
	}
	result := ToolFilter(AgentCompaction, tools)
	if len(result) != 0 {
		t.Errorf("deny all should return empty, got %d tools", len(result))
	}
}

func TestToolFilterDenySpecific(t *testing.T) {
	tools := []llm.ToolDef{
		{Name: "read", Parameters: map[string]any{"_tags": []string{"read"}}},
		{Name: "todowrite", Parameters: map[string]any{"_tags": []string{"write"}}},
		{Name: "bash", Parameters: map[string]any{"_tags": []string{"execute"}}},
	}
	result := ToolFilter(AgentGeneral, tools)
	for _, t2 := range result {
		if t2.Name == "todowrite" {
			t.Error("todowrite should be denied for general agent")
		}
	}
}

func TestToolFilterModeTags(t *testing.T) {
	tools := []llm.ToolDef{
		{Name: "read", Parameters: map[string]any{"_tags": []string{"read"}}},
		{Name: "write", Parameters: map[string]any{"_tags": []string{"write"}}},
		{Name: "bash", Parameters: map[string]any{"_tags": []string{"execute"}}},
	}

	result := ToolFilter(AgentExplore, tools)
	for _, t2 := range result {
		if t2.Name == "write" || t2.Name == "bash" {
			t.Errorf("explore agent should not have %q tool", t2.Name)
		}
	}
	found := false
	for _, t2 := range result {
		if t2.Name == "read" {
			found = true
		}
	}
	if !found {
		t.Error("explore agent should have read tool")
	}
}
