package tool

import (
	"context"
	"fmt"
	"strings"

	"github.com/morefun2602/opencode-go/internal/tools"
)

// TaskRunner is an interface matching Engine.CompleteTurn so we avoid import cycles.
type TaskRunner interface {
	CompleteTurn(ctx context.Context, workspaceID, sessionID, userText string) (string, error)
	CreateSession(ctx context.Context, workspaceID string) (string, error)
}

// SessionLookup checks if a session exists.
type SessionLookup interface {
	SessionExists(ctx context.Context, workspaceID, sessionID string) (bool, error)
}

type subagentKey struct{}

// SubagentNameFromContext returns the subagent name set by the task tool, if any.
func SubagentNameFromContext(ctx context.Context) (string, bool) {
	name, ok := ctx.Value(subagentKey{}).(string)
	return name, ok && name != ""
}

// WithSubagentContext injects subagent selection into context.
func WithSubagentContext(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, subagentKey{}, name)
}

// SubagentInfo describes a sub-agent available for the task tool.
type SubagentInfo struct {
	Name        string
	Description string
	CanUse      bool // false for hidden or non-subagent agents
}

// RegisterTask registers the task (sub-agent) tool. listSubagents provides
// available sub-agents; validateSubagent checks if a name is valid and usable.
func RegisterTask(
	reg *tools.Registry,
	runner TaskRunner,
	lookup SessionLookup,
	workspaceID string,
	maxDepth int,
	listSubagents func() []SubagentInfo,
	validateSubagent func(name string) (SubagentInfo, error),
) {
	type depthKey struct{}

	subDesc := func() string {
		subs := listSubagents()
		if len(subs) == 0 {
			return "no subagents available"
		}
		names := make([]string, 0, len(subs))
		for _, s := range subs {
			if s.CanUse {
				names = append(names, s.Name)
			}
		}
		return strings.Join(names, ", ")
	}

	reg.Register(tools.Tool{
		Name:        "task",
		Description: "Run a sub-agent to complete a task. Supports resuming previous tasks via task_id.",
		Tags:        []string{"execute"},
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"prompt":        map[string]any{"type": "string", "description": "task description for the sub-agent"},
				"task_id":       map[string]any{"type": "string", "description": "optional: resume an existing sub-agent session"},
				"subagent_type": map[string]any{"type": "string", "description": "optional: agent mode — " + subDesc()},
				"description":   map[string]any{"type": "string", "description": "optional: brief description for tracking"},
			},
			"required": []string{"prompt"},
		},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			depth, _ := ctx.Value(depthKey{}).(int)
			if depth >= maxDepth {
				return "", fmt.Errorf("exceeded max task nesting depth (%d)", maxDepth)
			}
			prompt := fmt.Sprint(args["prompt"])

			agentType, _ := args["subagent_type"].(string)
			agentName := "general"
			if agentType != "" {
				info, err := validateSubagent(agentType)
				if err != nil {
					return "", err
				}
				if !info.CanUse {
					return "", fmt.Errorf("agent %q cannot be used as subagent", agentType)
				}
				agentName = info.Name
			}

			var sid string
			taskID, _ := args["task_id"].(string)

			if taskID != "" {
				exists, err := lookup.SessionExists(ctx, workspaceID, taskID)
				if err != nil {
					return "", fmt.Errorf("checking task_id: %w", err)
				}
				if !exists {
					return "", fmt.Errorf("task_id %q not found", taskID)
				}
				sid = taskID
			} else {
				var err error
				sid, err = runner.CreateSession(ctx, workspaceID)
				if err != nil {
					return "", err
				}
			}

			sub := context.WithValue(ctx, depthKey{}, depth+1)
			sub = WithSubagentContext(sub, agentName)
			result, err := runner.CompleteTurn(sub, workspaceID, sid, prompt)
			if err != nil {
				return "", err
			}

			return fmt.Sprintf("task_id: %s\n%s", sid, result), nil
		},
	})
}
