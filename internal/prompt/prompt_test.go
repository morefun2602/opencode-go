package prompt

import (
	"strings"
	"testing"

	"github.com/morefun2602/opencode-go/internal/skill"
)

func TestModelPrompt(t *testing.T) {
	p := ModelPrompt("anthropic")
	if !strings.Contains(p, "Claude") {
		t.Errorf("anthropic prompt should mention Claude, got: %s", p[:50])
	}

	p = ModelPrompt("openai")
	if !strings.Contains(p, "coding assistant") {
		t.Errorf("openai prompt should mention coding assistant, got: %s", p[:50])
	}

	p = ModelPrompt("unknown")
	if p != openaiBasePrompt {
		t.Error("unknown provider should use openai base prompt")
	}
}

func TestEnvironmentPrompt(t *testing.T) {
	p := EnvironmentPrompt("/tmp/test-workspace")
	if !strings.Contains(p, "Working directory") {
		t.Error("should contain working directory")
	}
	if !strings.Contains(p, "Platform") {
		t.Error("should contain platform")
	}
	if !strings.Contains(p, "Date") {
		t.Error("should contain date")
	}
}

func TestSkillSummary(t *testing.T) {
	empty := SkillSummary(nil)
	if empty != "" {
		t.Error("empty skills should return empty string")
	}

	skills := []skill.Skill{
		{Name: "test-skill", Description: "A test skill"},
		{Name: "another", Description: "Another skill"},
	}
	s := SkillSummary(skills)
	if !strings.Contains(s, "test-skill") {
		t.Error("should contain skill name")
	}
	if !strings.Contains(s, "A test skill") {
		t.Error("should contain skill description")
	}
}

func TestBuild(t *testing.T) {
	s := Build(BuildOpts{
		ProviderType:  "anthropic",
		WorkspaceRoot: "/tmp/test",
	})
	if !strings.Contains(s, "Claude") {
		t.Error("should contain anthropic prompt")
	}
	if !strings.Contains(s, "Working directory") {
		t.Error("should contain environment info")
	}
}

func TestBuildWithAgentPrompt(t *testing.T) {
	s := Build(BuildOpts{
		ProviderType:  "openai",
		AgentPrompt:   "Custom agent prompt",
		WorkspaceRoot: "/tmp/test",
	})
	if !strings.Contains(s, "Custom agent prompt") {
		t.Error("should use agent prompt instead of model prompt")
	}
	if strings.Contains(s, "coding assistant") {
		t.Error("should not contain model prompt when agent prompt is set")
	}
}

func TestInstructionPromptEmpty(t *testing.T) {
	result := InstructionPrompt("/nonexistent/path/that/does/not/exist", nil)
	if result != "" {
		t.Errorf("should return empty for nonexistent path, got: %s", result)
	}
}
