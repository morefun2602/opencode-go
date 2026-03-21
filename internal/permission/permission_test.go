package permission

import (
	"testing"
)

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		pattern, target string
		want            bool
	}{
		{"*", "anything", true},
		{"*", "", true},
		{"bash", "bash", true},
		{"bash", "read", false},
		{"internal-*", "internal-debug", true},
		{"internal-*", "internal-", true},
		{"internal-*", "external-debug", false},
		{"read", "read", true},
	}
	for _, tt := range tests {
		if got := matchPattern(tt.pattern, tt.target); got != tt.want {
			t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
		}
	}
}

func TestToolPermissionName(t *testing.T) {
	for _, name := range []string{"edit", "write", "apply_patch", "multiedit"} {
		if got := ToolPermissionName(name); got != "edit" {
			t.Errorf("ToolPermissionName(%q) = %q, want %q", name, got, "edit")
		}
	}
	if got := ToolPermissionName("bash"); got != "bash" {
		t.Errorf("ToolPermissionName(%q) = %q, want %q", "bash", got, "bash")
	}
}

func TestDisabled_DenyAll(t *testing.T) {
	rs := Ruleset{{Permission: "*", Pattern: "*", Action: ActionDeny}}
	tools := []string{"read", "edit", "bash", "glob"}
	disabled := Disabled(tools, rs)
	for _, name := range tools {
		if !disabled[name] {
			t.Errorf("expected %q to be disabled", name)
		}
	}
}

func TestDisabled_DenyEditTools(t *testing.T) {
	rs := Ruleset{{Permission: "edit", Pattern: "*", Action: ActionDeny}}
	tools := []string{"edit", "write", "apply_patch", "multiedit", "read", "bash", "glob"}
	disabled := Disabled(tools, rs)

	for _, name := range []string{"edit", "write", "apply_patch", "multiedit"} {
		if !disabled[name] {
			t.Errorf("expected %q to be disabled", name)
		}
	}
	for _, name := range []string{"read", "bash", "glob"} {
		if disabled[name] {
			t.Errorf("expected %q to NOT be disabled", name)
		}
	}
}

func TestDisabled_OverrideRules(t *testing.T) {
	rs := Ruleset{
		{Permission: "*", Pattern: "*", Action: ActionDeny},
		{Permission: "read", Pattern: "*", Action: ActionAllow},
	}
	tools := []string{"read", "edit", "bash"}
	disabled := Disabled(tools, rs)

	if disabled["read"] {
		t.Error("read should be allowed (overridden)")
	}
	if !disabled["edit"] {
		t.Error("edit should be disabled")
	}
	if !disabled["bash"] {
		t.Error("bash should be disabled")
	}
}

func TestDisabled_PrefixPattern(t *testing.T) {
	rs := Ruleset{{Permission: "internal-*", Pattern: "*", Action: ActionDeny}}
	tools := []string{"internal-debug", "internal-trace", "read", "edit"}
	disabled := Disabled(tools, rs)

	if !disabled["internal-debug"] {
		t.Error("internal-debug should be disabled")
	}
	if !disabled["internal-trace"] {
		t.Error("internal-trace should be disabled")
	}
	if disabled["read"] {
		t.Error("read should not be disabled")
	}
}

func TestDisabled_EmptyRuleset(t *testing.T) {
	tools := []string{"read", "edit"}
	disabled := Disabled(tools, nil)
	if len(disabled) != 0 {
		t.Errorf("expected no disabled tools, got %v", disabled)
	}
}

func TestEvaluate(t *testing.T) {
	rs := Ruleset{
		{Permission: "*", Pattern: "*", Action: ActionDeny},
		{Permission: "bash", Pattern: "*", Action: ActionAsk},
		{Permission: "read", Pattern: "*", Action: ActionAllow},
	}

	if got := Evaluate("read", "file.go", rs); got != ActionAllow {
		t.Errorf("Evaluate(read) = %v, want allow", got)
	}
	if got := Evaluate("bash", "rm -rf /", rs); got != ActionAsk {
		t.Errorf("Evaluate(bash) = %v, want ask", got)
	}
	if got := Evaluate("edit", "file.go", rs); got != ActionDeny {
		t.Errorf("Evaluate(edit) = %v, want deny", got)
	}
}

func TestEvaluate_NoMatch(t *testing.T) {
	if got := Evaluate("read", "x", nil); got != ActionAllow {
		t.Errorf("Evaluate with nil ruleset = %v, want allow", got)
	}
}

func TestMerge(t *testing.T) {
	defaults := Ruleset{{Permission: "edit", Pattern: "*", Action: ActionDeny}}
	overrides := Ruleset{{Permission: "edit", Pattern: "*", Action: ActionAllow}}
	merged := Merge(defaults, overrides)

	if len(merged) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(merged))
	}
	if got := Evaluate("edit", "x", merged); got != ActionAllow {
		t.Errorf("merged Evaluate(edit) = %v, want allow (override)", got)
	}
}
