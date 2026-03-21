package permission

import "strings"

type Action string

const (
	ActionAllow Action = "allow"
	ActionDeny  Action = "deny"
	ActionAsk   Action = "ask"
)

type Rule struct {
	Permission string
	Pattern    string
	Action     Action
}

type Ruleset []Rule

var editTools = map[string]bool{
	"edit":        true,
	"write":       true,
	"apply_patch": true,
	"multiedit":   true,
}

// ToolPermissionName maps tool names to their permission category.
// Edit-family tools (edit, write, apply_patch, multiedit) all map to "edit".
func ToolPermissionName(toolName string) string {
	if editTools[toolName] {
		return "edit"
	}
	return toolName
}

func matchPattern(pattern, target string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(target, pattern[:len(pattern)-1])
	}
	return pattern == target
}

// Disabled returns the set of tool names that should be excluded based on the
// ruleset. For each tool, the last matching rule determines the action; tools
// with action "deny" are included in the returned set.
func Disabled(toolNames []string, ruleset Ruleset) map[string]bool {
	disabled := make(map[string]bool)
	for _, name := range toolNames {
		perm := ToolPermissionName(name)
		action := findLast(perm, name, ruleset)
		if action == ActionDeny {
			disabled[name] = true
		}
	}
	return disabled
}

// Evaluate returns the action for a given permission+target pair. The last
// matching rule wins; if no rule matches, ActionAllow is returned.
func Evaluate(perm, target string, ruleset Ruleset) Action {
	for i := len(ruleset) - 1; i >= 0; i-- {
		r := ruleset[i]
		if matchPattern(r.Permission, perm) && matchPattern(r.Pattern, target) {
			return r.Action
		}
	}
	return ActionAllow
}

// Merge concatenates two rulesets. Rules in overrides take precedence because
// findLast scans from the end.
func Merge(defaults, overrides Ruleset) Ruleset {
	out := make(Ruleset, 0, len(defaults)+len(overrides))
	out = append(out, defaults...)
	out = append(out, overrides...)
	return out
}

func findLast(perm, target string, ruleset Ruleset) Action {
	for i := len(ruleset) - 1; i >= 0; i-- {
		r := ruleset[i]
		if matchPattern(r.Permission, perm) && matchPattern(r.Pattern, target) {
			return r.Action
		}
	}
	return ActionAllow
}
