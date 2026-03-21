package runtime

import (
	"testing"

	"github.com/morefun2602/opencode-go/internal/llm"
	"github.com/morefun2602/opencode-go/internal/permission"
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

func TestListSubagents(t *testing.T) {
	subs := ListSubagents()
	if len(subs) == 0 {
		t.Fatal("should have at least one subagent")
	}
	names := make(map[string]bool)
	for _, a := range subs {
		names[a.Name] = true
		if a.Hidden {
			t.Errorf("subagent should not be hidden: %q", a.Name)
		}
		if !a.Subagent {
			t.Errorf("subagent flag should be true: %q", a.Name)
		}
	}
	if !names["explore"] {
		t.Error("explore should be a subagent")
	}
	if !names["general"] {
		t.Error("general should be a subagent")
	}
	if names["build"] {
		t.Error("build should NOT be a subagent")
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
		{Name: "read"},
		{Name: "todowrite"},
		{Name: "todoread"},
		{Name: "bash"},
	}
	result := ToolFilter(AgentGeneral, tools)
	for _, t2 := range result {
		if t2.Name == "todowrite" || t2.Name == "todoread" {
			t.Errorf("%q should be denied for general agent", t2.Name)
		}
	}
	if len(result) != 2 {
		t.Errorf("expected 2 tools (read, bash), got %d", len(result))
	}
}

func TestToolFilterExplorePermission(t *testing.T) {
	tools := []llm.ToolDef{
		{Name: "read"},
		{Name: "glob"},
		{Name: "grep"},
		{Name: "bash"},
		{Name: "edit"},
		{Name: "write"},
		{Name: "skill"},
	}
	result := ToolFilter(AgentExplore, tools)

	allowed := make(map[string]bool)
	for _, t2 := range result {
		allowed[t2.Name] = true
	}
	for _, want := range []string{"read", "glob", "grep", "bash", "skill"} {
		if !allowed[want] {
			t.Errorf("explore agent should have %q", want)
		}
	}
	for _, deny := range []string{"edit", "write"} {
		if allowed[deny] {
			t.Errorf("explore agent should NOT have %q", deny)
		}
	}
}

func TestToolFilterPlanPermission(t *testing.T) {
	tools := []llm.ToolDef{
		{Name: "read"},
		{Name: "glob"},
		{Name: "edit"},
		{Name: "write"},
		{Name: "apply_patch"},
		{Name: "bash"},
		{Name: "plan_enter"},
	}
	result := ToolFilter(AgentPlan, tools)

	allowed := make(map[string]bool)
	for _, t2 := range result {
		allowed[t2.Name] = true
	}
	if allowed["edit"] || allowed["write"] || allowed["apply_patch"] {
		t.Error("plan agent should deny edit tools")
	}
	if allowed["bash"] {
		t.Error("plan agent should deny bash")
	}
	if !allowed["read"] {
		t.Error("plan agent should allow read")
	}
}

func TestToolFilterModeTagsFallback(t *testing.T) {
	agent := Agent{
		Name: "legacy",
		Mode: Mode{Name: "test", Tags: []string{"read"}},
	}
	tools := []llm.ToolDef{
		{Name: "read", Parameters: map[string]any{"_tags": []string{"read"}}},
		{Name: "write", Parameters: map[string]any{"_tags": []string{"write"}}},
	}
	result := ToolFilter(agent, tools)
	if len(result) != 1 || result[0].Name != "read" {
		t.Errorf("fallback to Mode Tags should keep only read, got %v", result)
	}
}

func TestToolFilterPermissionDenyEdit(t *testing.T) {
	agent := Agent{
		Name: "custom",
		Permission: permission.Ruleset{
			{Permission: "edit", Pattern: "*", Action: permission.ActionDeny},
		},
	}
	tools := []llm.ToolDef{
		{Name: "read"},
		{Name: "edit"},
		{Name: "write"},
		{Name: "multiedit"},
		{Name: "apply_patch"},
		{Name: "bash"},
	}
	result := ToolFilter(agent, tools)
	allowed := make(map[string]bool)
	for _, t2 := range result {
		allowed[t2.Name] = true
	}
	if allowed["edit"] || allowed["write"] || allowed["multiedit"] || allowed["apply_patch"] {
		t.Error("deny edit should block all edit-family tools")
	}
	if !allowed["read"] || !allowed["bash"] {
		t.Error("non-edit tools should be allowed")
	}
}

func TestRegisterAgent(t *testing.T) {
	RegisterAgent(Agent{
		Name:        "test-custom",
		Description: "test agent",
		Subagent:    true,
	})
	defer func() { delete(agentRegistry, "test-custom") }()

	a, ok := GetAgent("test-custom")
	if !ok {
		t.Fatal("registered agent should be found")
	}
	if a.Description != "test agent" {
		t.Errorf("expected 'test agent', got %q", a.Description)
	}
}
