package tool

import (
	"context"
	"fmt"
	"strings"

	"github.com/morefun2602/opencode-go/internal/skill"
	"github.com/morefun2602/opencode-go/internal/tools"
)

func registerSkillTool(reg *tools.Registry, skills []skill.Skill) {
	reg.Register(tools.Tool{
		Name:        "skill",
		Description: "List available skills or load a specific skill's instructions",
		Tags:        []string{"read"},
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{"type": "string", "description": "skill name to load (omit to list all)"},
			},
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

			for _, s := range skills {
				if strings.EqualFold(s.Name, name) {
					return s.Body, nil
				}
			}
			var names []string
			for _, s := range skills {
				names = append(names, s.Name)
			}
			return "", fmt.Errorf("skill %q not found. Available: %s", name, strings.Join(names, ", "))
		},
	})
}
