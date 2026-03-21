package tool

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/morefun2602/opencode-go/internal/skill"
	"github.com/morefun2602/opencode-go/internal/tools"
)

func registerSkillTool(reg *tools.Registry, skills []skill.Skill) {
	description := buildSkillDescription(skills)
	nameHint := buildNameHint(skills)

	reg.Register(tools.Tool{
		Name:        "skill",
		Description: description,
		Tags:        []string{"read"},
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": fmt.Sprintf("The name of the skill from available_skills%s", nameHint),
				},
			},
			"required": []string{"name"},
		},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			name, _ := args["name"].(string)

			if name == "" {
				if len(skills) == 0 {
					return "no skills available", nil
				}
				var sb strings.Builder
				sb.WriteString("Available skills:\n")
				for _, s := range skills {
					desc := s.Description
					if desc == "" && len(s.Body) > 80 {
						desc = s.Body[:80] + "..."
					} else if desc == "" {
						desc = s.Body
					}
					fmt.Fprintf(&sb, "- %s: %s\n", s.Name, desc)
				}
				return sb.String(), nil
			}

			var found *skill.Skill
			for i := range skills {
				if strings.EqualFold(skills[i].Name, name) {
					found = &skills[i]
					break
				}
			}
			if found == nil {
				var names []string
				for _, s := range skills {
					names = append(names, s.Name)
				}
				return "", fmt.Errorf("skill %q not found. Available: %s", name, strings.Join(names, ", "))
			}

			dir := filepath.Dir(found.Path)
			base := (&url.URL{Scheme: "file", Path: dir}).String()
			files := collectSkillFiles(dir, 10)

			var sb strings.Builder
			fmt.Fprintf(&sb, "<skill_content name=%q>\n", found.Name)
			fmt.Fprintf(&sb, "# Skill: %s\n\n", found.Name)
			sb.WriteString(strings.TrimSpace(found.Body))
			sb.WriteString("\n\n")
			fmt.Fprintf(&sb, "Base directory for this skill: %s\n", base)
			sb.WriteString("Relative paths in this skill (e.g., scripts/, reference/) are relative to this base directory.\n")
			sb.WriteString("Note: file list is sampled.\n\n")
			sb.WriteString("<skill_files>\n")
			for _, f := range files {
				fmt.Fprintf(&sb, "<file>%s</file>\n", f)
			}
			sb.WriteString("</skill_files>\n")
			sb.WriteString("</skill_content>")
			return sb.String(), nil
		},
	})
}

func buildSkillDescription(skills []skill.Skill) string {
	if len(skills) == 0 {
		return "Load a specialized skill that provides domain-specific instructions and workflows. No skills are currently available."
	}
	return strings.Join([]string{
		"Load a specialized skill that provides domain-specific instructions and workflows.",
		"",
		"When you recognize that a task matches one of the available skills listed below, use this tool to load the full skill instructions.",
		"",
		"The skill will inject detailed instructions, workflows, and access to bundled resources (scripts, references, templates) into the conversation context.",
		"",
		`Tool output includes a <skill_content name="..."> block with the loaded content.`,
		"",
		"The following skills provide specialized sets of instructions for particular tasks",
		"Invoke this tool to load a skill when a task matches one of the available skills listed below:",
		"",
		skill.Fmt(skills, false),
	}, "\n")
}

func buildNameHint(skills []skill.Skill) string {
	if len(skills) == 0 {
		return ""
	}
	limit := 3
	if len(skills) < limit {
		limit = len(skills)
	}
	examples := make([]string, limit)
	for i := 0; i < limit; i++ {
		examples[i] = "'" + skills[i].Name + "'"
	}
	return fmt.Sprintf(" (e.g., %s, ...)", strings.Join(examples, ", "))
}

func collectSkillFiles(dir string, limit int) []string {
	var files []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if strings.EqualFold(info.Name(), "SKILL.md") {
			return nil
		}
		files = append(files, path)
		if len(files) >= limit {
			return filepath.SkipAll
		}
		return nil
	})
	return files
}
