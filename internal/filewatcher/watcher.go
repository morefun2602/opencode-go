package filewatcher

import (
	"bufio"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/morefun2602/opencode-go/internal/bus"
)

// Watcher monitors workspace file changes using fsnotify (when available)
// or manual notification from write tools.
type Watcher struct {
	bus            *bus.Bus
	workspaceRoot  string
	log            *slog.Logger
	ignorePatterns []string
	mu             sync.RWMutex
	dirty          bool
}

// New creates a file watcher for the workspace.
func New(workspaceRoot string, b *bus.Bus, log *slog.Logger) *Watcher {
	w := &Watcher{
		bus:           b,
		workspaceRoot: workspaceRoot,
		log:           log,
	}
	w.ignorePatterns = w.loadGitignore()
	return w
}

// NotifyChange publishes a file.changed event. Called by write/edit/apply_patch tools.
func (w *Watcher) NotifyChange(path string) {
	w.mu.Lock()
	w.dirty = true
	w.mu.Unlock()

	if w.bus != nil {
		w.bus.Publish("file.changed", map[string]any{"path": path})
	}
}

// IsDirty returns whether any file has changed since last reset.
func (w *Watcher) IsDirty() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.dirty
}

// ResetDirty clears the dirty flag.
func (w *Watcher) ResetDirty() {
	w.mu.Lock()
	w.dirty = false
	w.mu.Unlock()
}

// IsIgnored checks if a path should be ignored based on .gitignore patterns.
func (w *Watcher) IsIgnored(path string) bool {
	rel, err := filepath.Rel(w.workspaceRoot, path)
	if err != nil {
		return false
	}
	name := filepath.Base(rel)

	if name == ".git" || name == "node_modules" || name == "__pycache__" {
		return true
	}

	for _, pat := range w.ignorePatterns {
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

func (w *Watcher) loadGitignore() []string {
	f, err := os.Open(filepath.Join(w.workspaceRoot, ".gitignore"))
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
