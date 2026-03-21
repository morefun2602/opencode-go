package skill

import (
	"os"
	"path/filepath"
	"strings"
)

// Skill 从目录加载的简易技能块（元数据+正文）。
type Skill struct {
	Name        string
	Description string
	Body        string
	Path        string
}

// LoadDir 读取目录下 `*.md` 为技能正文，文件名为 Name。
func LoadDir(dir string) ([]Skill, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Skill
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if !strings.HasSuffix(strings.ToLower(n), ".md") {
			continue
		}
		p := filepath.Join(dir, n)
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		base := strings.TrimSuffix(n, filepath.Ext(n))
		s := Skill{Name: base, Body: string(b), Path: p}
		s.Name, s.Description, s.Body = parseFrontmatter(string(b), base)
		out = append(out, s)
	}
	return out, nil
}

// DiscoverSkills searches multiple paths for SKILL.md files recursively.
func DiscoverSkills(searchPaths []string) ([]Skill, error) {
	seen := map[string]bool{}
	var result []Skill
	for _, base := range searchPaths {
		err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			name := info.Name()
			if !strings.EqualFold(name, "SKILL.md") && !strings.HasSuffix(strings.ToLower(name), ".md") {
				return nil
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			dir := filepath.Base(filepath.Dir(path))
			skillName, desc, body := parseFrontmatter(string(b), dir)
			if seen[skillName] {
				return nil
			}
			seen[skillName] = true
			result = append(result, Skill{
				Name:        skillName,
				Description: desc,
				Body:        body,
				Path:        path,
			})
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			continue
		}
	}
	return result, nil
}

// parseFrontmatter extracts YAML frontmatter name/description if present.
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

// InjectPrompt 将技能正文拼入系统提示前缀（边界：不修改工具列表，仅文本）。
func InjectPrompt(base string, skills []Skill) string {
	if len(skills) == 0 {
		return base
	}
	var sb strings.Builder
	sb.WriteString(base)
	sb.WriteString("\n\n## Skills\n")
	for _, s := range skills {
		sb.WriteString("\n### ")
		sb.WriteString(s.Name)
		sb.WriteString("\n")
		sb.WriteString(s.Body)
	}
	return sb.String()
}
