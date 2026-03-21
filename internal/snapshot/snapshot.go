package snapshot

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// Service provides workspace snapshot capabilities using git.
type Service struct {
	workspaceRoot string
	snapshotDir   string
	log           *slog.Logger
	mu            sync.Mutex
	available     bool
}

// New creates a snapshot service. Returns a service that gracefully degrades
// if the workspace is not a git repository.
func New(workspaceRoot string, log *slog.Logger) *Service {
	s := &Service{
		workspaceRoot: workspaceRoot,
		log:           log,
	}
	if isGitRepo(workspaceRoot) {
		s.available = true
		s.snapshotDir = filepath.Join(workspaceRoot, ".opencode", "snapshots")
		_ = os.MkdirAll(s.snapshotDir, 0o755)
	} else if log != nil {
		log.Warn("snapshot: workspace is not a git repo, snapshots disabled", "path", workspaceRoot)
	}
	return s
}

func isGitRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

// Track saves the current workspace state for the given session/step.
func (s *Service) Track(ctx context.Context, sessionID, stepID string) error {
	if !s.available {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	diff, err := s.gitDiff(ctx)
	if err != nil {
		return fmt.Errorf("snapshot track: %w", err)
	}

	key := sessionID + "/" + stepID
	path := filepath.Join(s.snapshotDir, sanitizeKey(key)+".diff")
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	return os.WriteFile(path, []byte(diff), 0o644)
}

// Patch records the diff between last track and now for the given step.
func (s *Service) Patch(ctx context.Context, sessionID, stepID string) error {
	return s.Track(ctx, sessionID, stepID+"-post")
}

// Restore applies a previously saved snapshot.
func (s *Service) Restore(ctx context.Context, sessionID, stepID string) error {
	if !s.available {
		return fmt.Errorf("snapshots not available")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	key := sessionID + "/" + stepID
	path := filepath.Join(s.snapshotDir, sanitizeKey(key)+".diff")
	diff, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("snapshot not found: %w", err)
	}

	cmd := exec.CommandContext(ctx, "git", "checkout", ".")
	cmd.Dir = s.workspaceRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout: %s: %w", out, err)
	}

	if len(diff) > 0 {
		cmd = exec.CommandContext(ctx, "git", "apply", "--allow-empty")
		cmd.Dir = s.workspaceRoot
		cmd.Stdin = strings.NewReader(string(diff))
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git apply: %s: %w", out, err)
		}
	}
	return nil
}

// Diff returns the unified diff between two snapshots.
func (s *Service) Diff(ctx context.Context, sessionID, stepA, stepB string) (string, error) {
	if !s.available {
		return "", fmt.Errorf("snapshots not available")
	}
	pathA := filepath.Join(s.snapshotDir, sanitizeKey(sessionID+"/"+stepA)+".diff")
	pathB := filepath.Join(s.snapshotDir, sanitizeKey(sessionID+"/"+stepB)+".diff")

	a, err := os.ReadFile(pathA)
	if err != nil {
		return "", fmt.Errorf("snapshot A not found: %w", err)
	}
	b, err := os.ReadFile(pathB)
	if err != nil {
		return "", fmt.Errorf("snapshot B not found: %w", err)
	}

	if string(a) == string(b) {
		return "no changes", nil
	}
	return fmt.Sprintf("--- snapshot %s\n+++ snapshot %s\n%s", stepA, stepB, string(b)), nil
}

// Available reports whether snapshots are functional.
func (s *Service) Available() bool { return s.available }

func (s *Service) gitDiff(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "HEAD")
	cmd.Dir = s.workspaceRoot
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func sanitizeKey(key string) string {
	return strings.ReplaceAll(key, "/", "_")
}
