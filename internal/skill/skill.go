package skill

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Skill struct {
	Name        string
	Description string
	Body        string
	Path        string
	Location    string
}

// DiscoverSkills searches multiple paths for SKILL.md files recursively.
// Only files named SKILL.md (case-insensitive) are matched.
// Earlier search paths take priority: if a skill name is already seen, duplicates are skipped with a warning.
func DiscoverSkills(searchPaths []string, log *slog.Logger) ([]Skill, error) {
	seen := map[string]string{} // skillName -> first seen path
	var result []Skill
	for _, base := range searchPaths {
		err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			if !strings.EqualFold(info.Name(), "SKILL.md") {
				return nil
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			absPath, _ := filepath.Abs(path)
			if absPath == "" {
				absPath = path
			}
			dir := filepath.Base(filepath.Dir(path))
			skillName, desc, body := parseFrontmatter(string(b), dir)
			if existing, ok := seen[skillName]; ok {
				if log != nil {
					log.Warn("duplicate skill name, using first found",
						"name", skillName,
						"existing", existing,
						"duplicate", absPath,
					)
				}
				return nil
			}
			seen[skillName] = absPath
			result = append(result, Skill{
				Name:        skillName,
				Description: desc,
				Body:        body,
				Path:        absPath,
				Location:    absPath,
			})
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			continue
		}
	}
	return result, nil
}

func parseFrontmatter(content, defaultName string) (name, description, body string) {
	name = defaultName
	body = content
	if !strings.HasPrefix(content, "---") {
		return
	}
	end := strings.Index(content[3:], "---")
	if end < 0 {
		return
	}
	fm := content[3 : 3+end]
	body = strings.TrimSpace(content[3+end+3:])
	for _, line := range strings.Split(fm, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			v := strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			v = strings.Trim(v, "\"'")
			if v != "" {
				name = v
			}
		}
		if strings.HasPrefix(line, "description:") {
			v := strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			v = strings.Trim(v, "\"'")
			description = v
		}
	}
	return
}

// Fmt formats a skill list in verbose (XML) or concise (Markdown) mode.
func Fmt(skills []Skill, verbose bool) string {
	if len(skills) == 0 {
		return "No skills are currently available."
	}
	if verbose {
		var lines []string
		lines = append(lines, "<available_skills>")
		for _, s := range skills {
			loc := (&url.URL{Scheme: "file", Path: s.Location}).String()
			lines = append(lines,
				"  <skill>",
				fmt.Sprintf("    <name>%s</name>", s.Name),
				fmt.Sprintf("    <description>%s</description>", s.Description),
				fmt.Sprintf("    <location>%s</location>", loc),
				"  </skill>",
			)
		}
		lines = append(lines, "</available_skills>")
		return strings.Join(lines, "\n")
	}
	var lines []string
	lines = append(lines, "## Available Skills")
	for _, s := range skills {
		lines = append(lines, fmt.Sprintf("- **%s**: %s", s.Name, s.Description))
	}
	return strings.Join(lines, "\n")
}
