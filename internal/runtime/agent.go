package runtime

import "github.com/morefun2602/opencode-go/internal/llm"

type ToolPermission struct {
	Deny  []string
	Allow []string
}

type Agent struct {
	Name            string
	Prompt          string
	Mode            Mode
	Hidden          bool
	ToolPermissions ToolPermission
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

var (
	AgentBuild = Agent{
		Name: "build",
		Mode: ModeBuild,
	}

	AgentPlan = Agent{
		Name: "plan",
		Mode: ModePlan,
	}

	AgentExplore = Agent{
		Name:   "explore",
		Prompt: promptExplore,
		Mode:   ModeExplore,
	}

	AgentGeneral = Agent{
		Name: "general",
		Mode: ModeBuild,
		ToolPermissions: ToolPermission{
			Deny: []string{"todowrite"},
		},
	}

	AgentCompaction = Agent{
		Name:   "compaction",
		Prompt: promptCompaction,
		Mode:   Mode{Name: "compaction"},
		Hidden: true,
		ToolPermissions: ToolPermission{
			Deny: []string{"*"},
		},
	}

	AgentTitle = Agent{
		Name:   "title",
		Prompt: promptTitle,
		Mode:   Mode{Name: "title"},
		Hidden: true,
		ToolPermissions: ToolPermission{
			Deny: []string{"*"},
		},
	}

	AgentSummary = Agent{
		Name:   "summary",
		Prompt: promptSummary,
		Mode:   Mode{Name: "summary"},
		Hidden: true,
		ToolPermissions: ToolPermission{
			Deny: []string{"*"},
		},
	}
)

var builtinAgents = map[string]Agent{
	"build":      AgentBuild,
	"plan":       AgentPlan,
	"explore":    AgentExplore,
	"general":    AgentGeneral,
	"compaction": AgentCompaction,
	"title":      AgentTitle,
	"summary":    AgentSummary,
}

func GetAgent(name string) (Agent, bool) {
	a, ok := builtinAgents[name]
	return a, ok
}

func ListAgents() []Agent {
	out := make([]Agent, 0, len(builtinAgents))
	for _, a := range builtinAgents {
		if !a.Hidden {
			out = append(out, a)
		}
	}
	return out
}

func ToolFilter(agent Agent, allTools []llm.ToolDef) []llm.ToolDef {
	perm := agent.ToolPermissions

	if len(perm.Deny) == 1 && perm.Deny[0] == "*" {
		return nil
	}

	modeTags := agent.Mode.Tags
	denySet := make(map[string]bool, len(perm.Deny))
	for _, d := range perm.Deny {
		denySet[d] = true
	}

	allowSet := make(map[string]bool, len(perm.Allow))
	for _, a := range perm.Allow {
		allowSet[a] = true
	}

	var out []llm.ToolDef
	for _, t := range allTools {
		if denySet[t.Name] {
			continue
		}

		if len(allowSet) > 0 {
			if !allowSet[t.Name] {
				continue
			}
		}

		if len(modeTags) > 0 {
			tags := extractTags(t)
			if len(tags) > 0 && !hasOverlap(tags, modeTags) {
				continue
			}
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
