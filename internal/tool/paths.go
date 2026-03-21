package tool

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveUnder 将相对路径限制在 root 之下（越界则错误）。
func ResolveUnder(root, p string) (string, error) {
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	clean := filepath.Clean(p)
	if filepath.IsAbs(clean) {
		if !strings.HasPrefix(clean, absRoot+string(os.PathSeparator)) && clean != absRoot {
			return "", fmt.Errorf("path outside workspace")
		}
		return clean, nil
	}
	joined := filepath.Join(absRoot, clean)
	joinedAbs, err := filepath.Abs(joined)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(joinedAbs, absRoot+string(os.PathSeparator)) && joinedAbs != absRoot {
		return "", fmt.Errorf("path outside workspace")
	}
	return joinedAbs, nil
}
