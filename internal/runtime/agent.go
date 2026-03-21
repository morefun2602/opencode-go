package runtime

import (
	"github.com/morefun2602/opencode-go/internal/llm"
	"github.com/morefun2602/opencode-go/internal/permission"
)

type Agent struct {
	Name        string
	Description string
	Prompt      string
	Mode        Mode
	Hidden      bool
	Subagent    bool
	Steps       int
	Model       string
	Temperature *float64
	Permission  permission.Ruleset
}

const promptExplore = `You are an exploration agent. Your job is to find information in the codebase.

Focus on using glob, grep, and read tools to explore the codebase efficiently. Do NOT make any changes to files. Start with broad searches and narrow down.

Guidelines:
- Use glob to find files by name patterns
- Use grep to search for code patterns
- Use read to examine file contents
- Do NOT use write, edit, or bash tools`

const promptCompaction = `Summarize the conversation so far. Be thorough but concise. Include:

## Goal
What the user is trying to accomplish.

## Key Decisions
Important decisions made during the conversation.

## Progress
What has been accomplished so far, including specific files modified.

## Relevant Files
List files that were read, created, or modified.

## Open Issues
Any unresolved problems or next steps.`

const promptTitle = `Generate a concise title (maximum 50 characters) for the following conversation. Return ONLY the title text, nothing else. No quotes, no explanation.`

const promptSummary = `Generate a brief summary (2-3 sentences) of what was accomplished in this conversation, similar to a pull request description. Focus on the changes made and their purpose. Return ONLY the summary text.`

var denyAll = permission.Ruleset{
	{Permission: "*", Pattern: "*", Action: permission.ActionDeny},
}

var (
	AgentBuild = Agent{
		Name:        "build",
		Description: "Default agent with full tool access",
		Mode:        ModeBuild,
	}

	AgentPlan = Agent{
		Name:        "plan",
		Description: "Read-only planning mode, no code changes",
		Mode:        ModePlan,
		Permission: permission.Ruleset{
			{Permission: "edit", Pattern: "*", Action: permission.ActionDeny},
			{Permission: "bash", Pattern: "*", Action: permission.ActionDeny},
		},
	}

	AgentExplore = Agent{
		Name:        "explore",
		Description: "Code exploration agent, read-only with bash",
		Prompt:      promptExplore,
		Mode:        ModeExplore,
		Subagent:    true,
		Permission: permission.Ruleset{
			{Permission: "*", Pattern: "*", Action: permission.ActionDeny},
			{Permission: "read", Pattern: "*", Action: permission.ActionAllow},
			{Permission: "glob", Pattern: "*", Action: permission.ActionAllow},
			{Permission: "grep", Pattern: "*", Action: permission.ActionAllow},
			{Permission: "bash", Pattern: "*", Action: permission.ActionAllow},
			{Permission: "skill", Pattern: "*", Action: permission.ActionAllow},
			{Permission: "ls", Pattern: "*", Action: permission.ActionAllow},
			{Permission: "webfetch", Pattern: "*", Action: permission.ActionAllow},
			{Permission: "websearch", Pattern: "*", Action: permission.ActionAllow},
		},
	}

	AgentGeneral = Agent{
		Name:        "general",
		Description: "General-purpose sub-agent for multi-step tasks",
		Mode:        ModeBuild,
		Subagent:    true,
		Permission: permission.Ruleset{
			{Permission: "todowrite", Pattern: "*", Action: permission.ActionDeny},
			{Permission: "todoread", Pattern: "*", Action: permission.ActionDeny},
		},
	}

	AgentCompaction = Agent{
		Name:        "compaction",
		Description: "Conversation summarization",
		Prompt:      promptCompaction,
		Mode:        Mode{Name: "compaction"},
		Hidden:      true,
		Permission:  denyAll,
	}

	AgentTitle = Agent{
		Name:        "title",
		Description: "Session title generation",
		Prompt:      promptTitle,
		Mode:        Mode{Name: "title"},
		Hidden:      true,
		Permission:  denyAll,
	}

	AgentSummary = Agent{
		Name:        "summary",
		Description: "Session summary generation",
		Prompt:      promptSummary,
		Mode:        Mode{Name: "summary"},
		Hidden:      true,
		Permission:  denyAll,
	}
)

var agentRegistry = map[string]Agent{
	"build":      AgentBuild,
	"plan":       AgentPlan,
	"explore":    AgentExplore,
	"general":    AgentGeneral,
	"compaction": AgentCompaction,
	"title":      AgentTitle,
	"summary":    AgentSummary,
}

func RegisterAgent(a Agent) {
	agentRegistry[a.Name] = a
}

func GetAgent(name string) (Agent, bool) {
	a, ok := agentRegistry[name]
	return a, ok
}

func ListAgents() []Agent {
	out := make([]Agent, 0, len(agentRegistry))
	for _, a := range agentRegistry {
		if !a.Hidden {
			out = append(out, a)
		}
	}
	return out
}

// ListSubagents returns agents that can be used via the task tool.
func ListSubagents() []Agent {
	out := make([]Agent, 0)
	for _, a := range agentRegistry {
		if !a.Hidden && a.Subagent {
			out = append(out, a)
		}
	}
	return out
}

// ToolFilter returns the subset of allTools that the given agent is allowed to
// use. When the agent has a Permission Ruleset, it takes precedence. Otherwise
// the legacy Mode Tags filtering is used as a fallback.
func ToolFilter(agent Agent, allTools []llm.ToolDef) []llm.ToolDef {
	if len(agent.Permission) > 0 {
		names := make([]string, len(allTools))
		for i, t := range allTools {
			names[i] = t.Name
		}
		disabled := permission.Disabled(names, agent.Permission)
		var out []llm.ToolDef
		for _, t := range allTools {
			if !disabled[t.Name] {
				out = append(out, t)
			}
		}
		return out
	}

	modeTags := agent.Mode.Tags
	if len(modeTags) == 0 {
		return allTools
	}

	var out []llm.ToolDef
	for _, t := range allTools {
		tags := extractTags(t)
		if len(tags) > 0 && !hasOverlap(tags, modeTags) {
			continue
		}
		out = append(out, t)
	}
	return out
}

func extractTags(t llm.ToolDef) []string {
	params, ok := t.Parameters["_tags"]
	if !ok {
		return nil
	}
	if arr, ok := params.([]string); ok {
		return arr
	}
	if arr, ok := params.([]any); ok {
		tags := make([]string, 0, len(arr))
		for _, v := range arr {
			if s, ok := v.(string); ok {
				tags = append(tags, s)
			}
		}
		return tags
	}
	return nil
}
