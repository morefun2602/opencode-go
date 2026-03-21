package tool

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/morefun2602/opencode-go/internal/tools"
)

func registerLs(reg *tools.Registry, root string) {
	reg.Register(tools.Tool{
		Name:        "ls",
		Description: "List directory tree structure, respecting .gitignore patterns",
		Tags:        []string{"read"},
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string", "description": "directory path (default: workspace root)"},
			},
		},
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			p, _ := args["path"].(string)
			if p == "" {
				p = "."
			}
			rp, err := ResolveUnder(root, p)
			if err != nil {
				return "", err
			}
			info, err := os.Stat(rp)
			if err != nil {
				return "", err
			}
			if !info.IsDir() {
				return "", fmt.Errorf("%s is not a directory", p)
			}

			ignorePatterns := loadGitignore(rp)
			var sb strings.Builder
			err = walkTree(rp, rp, "", &sb, ignorePatterns, 0, 3)
			if err != nil {
				return "", err
			}
			return sb.String(), nil
		},
	})
}

func walkTree(base, dir, prefix string, sb *strings.Builder, ignore []string, depth, maxDepth int) error {
	if depth > maxDepth {
		fmt.Fprintf(sb, "%s...\n", prefix)
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var visible []os.DirEntry
	for _, e := range entries {
		name := e.Name()
		if name == ".git" || name == "node_modules" || name == "__pycache__" {
			continue
		}
		rel, _ := filepath.Rel(base, filepath.Join(dir, name))
		if isIgnored(rel, e.IsDir(), ignore) {
			continue
		}
		visible = append(visible, e)
	}
	for i, e := range visible {
		connector := "├── "
		childPrefix := prefix + "│   "
		if i == len(visible)-1 {
			connector = "└── "
			childPrefix = prefix + "    "
		}
		fmt.Fprintf(sb, "%s%s%s\n", prefix, connector, e.Name())
		if e.IsDir() {
			_ = walkTree(base, filepath.Join(dir, e.Name()), childPrefix, sb, ignore, depth+1, maxDepth)
		}
	}
	return nil
}

func loadGitignore(dir string) []string {
	f, err := os.Open(filepath.Join(dir, ".gitignore"))
	if err != nil {
		return nil
	}
	defer f.Close()
	var patterns []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns
}

func isIgnored(rel string, isDir bool, patterns []string) bool {
	name := filepath.Base(rel)
	for _, pat := range patterns {
		pat = strings.TrimSuffix(pat, "/")
		if matched, _ := filepath.Match(pat, name); matched {
			return true
		}
		if matched, _ := filepath.Match(pat, rel); matched {
			return true
		}
	}
	return false
}
