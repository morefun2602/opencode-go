package prompt

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/morefun2602/opencode-go/internal/skill"
)

const openaiBasePrompt = `You are a helpful coding assistant. You have access to tools that let you read, write, and search files, run commands, and more. Use them to help the user with their coding tasks. Be concise and precise in your responses. When making changes, explain what you're doing briefly.`

const anthropicBasePrompt = `You are Claude, an AI assistant created by Anthropic. You are a coding agent that helps users with software engineering tasks. You have access to a set of tools that let you interact with the codebase, run commands, and more. Be concise and direct. When making changes, keep them minimal and focused.`

func ModelPrompt(providerType string) string {
	switch providerType {
	case "anthropic":
		return anthropicBasePrompt
	case "openai":
		return openaiBasePrompt
	default:
		return openaiBasePrompt
	}
}

func EnvironmentPrompt(workspaceRoot string) string {
	var sb strings.Builder
	sb.WriteString("## Environment\n\n")

	abs, err := filepath.Abs(workspaceRoot)
	if err == nil {
		workspaceRoot = abs
	}
	fmt.Fprintf(&sb, "- Working directory: %s\n", workspaceRoot)
	fmt.Fprintf(&sb, "- Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(&sb, "- Date: %s\n", time.Now().Format("2006-01-02"))

	branch := gitBranch(workspaceRoot)
	if branch != "" {
		fmt.Fprintf(&sb, "- Git branch: %s\n", branch)
	}

	return sb.String()
}

func gitBranch(dir string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

var instructionFiles = []string{"AGENTS.md", "CLAUDE.md", "CONTEXT.md"}

func InstructionPrompt(workspaceRoot string, configInstructions []string) string {
	var parts []string

	for _, name := range instructionFiles {
		content := findUp(workspaceRoot, name)
		if content != "" {
			parts = append(parts, content)
		}
	}

	for _, instr := range configInstructions {
		if strings.HasPrefix(instr, "http://") || strings.HasPrefix(instr, "https://") {
			content := fetchURL(instr)
			if content != "" {
				parts = append(parts, content)
			}
		} else {
			p := instr
			if !filepath.IsAbs(p) {
				p = filepath.Join(workspaceRoot, p)
			}
			b, err := os.ReadFile(p)
			if err == nil {
				parts = append(parts, string(b))
			}
		}
	}

	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n\n")
}

func findUp(dir string, name string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	for {
		p := filepath.Join(abs, name)
		if b, err := os.ReadFile(p); err == nil {
			return string(b)
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			break
		}
		abs = parent
	}
	return ""
}

func fetchURL(url string) string {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ""
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return ""
	}
	return string(b)
}

func SkillSummary(skills []skill.Skill) string {
	if len(skills) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("## Available Skills\n\n")
	for _, s := range skills {
		fmt.Fprintf(&sb, "- **%s**", s.Name)
		if s.Description != "" {
			fmt.Fprintf(&sb, ": %s", s.Description)
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\nUse the `skill` tool to load full instructions for any skill.\n")
	return sb.String()
}

type BuildOpts struct {
	ProviderType     string
	AgentPrompt      string
	WorkspaceRoot    string
	ConfigInstructions []string
	Skills           []skill.Skill
}

func Build(opts BuildOpts) string {
	var parts []string

	if opts.AgentPrompt != "" {
		parts = append(parts, opts.AgentPrompt)
	} else {
		parts = append(parts, ModelPrompt(opts.ProviderType))
	}

	env := EnvironmentPrompt(opts.WorkspaceRoot)
	if env != "" {
		parts = append(parts, env)
	}

	instr := InstructionPrompt(opts.WorkspaceRoot, opts.ConfigInstructions)
	if instr != "" {
		parts = append(parts, instr)
	}

	skillSum := SkillSummary(opts.Skills)
	if skillSum != "" {
		parts = append(parts, skillSum)
	}

	return strings.Join(parts, "\n\n")
}
