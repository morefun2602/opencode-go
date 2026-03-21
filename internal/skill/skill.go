package skill

import (
	"os"
	"path/filepath"
	"strings"
)

// Skill 从目录加载的简易技能块（元数据+正文）。
type Skill struct {
	Name string
	Body string
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
		b, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			return nil, err
		}
		base := strings.TrimSuffix(n, filepath.Ext(n))
		out = append(out, Skill{Name: base, Body: string(b)})
	}
	return out, nil
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
