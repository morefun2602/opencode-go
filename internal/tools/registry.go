package tools

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/morefun2602/opencode-go/internal/truncate"
)

// Fn 工具实现：args 已由 schema 校验。
type Fn func(ctx context.Context, args map[string]any) (string, error)

// Tool 注册项。
type Tool struct {
	Name        string
	Description string
	Schema      map[string]any // JSON Schema or simple key-value
	Tags        []string
	Fn          Fn
}

// Registry 工具注册表；失败时返回结构化错误，不静默吞掉。
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
	log   *slog.Logger
}

func New(log *slog.Logger) *Registry {
	return &Registry{tools: map[string]Tool{}, log: log}
}

func (r *Registry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name] = t
}

// Has 报告是否已注册内置工具。
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.tools[name]
	return ok
}

// Get returns a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// List returns all registered tools.
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

func (r *Registry) Run(ctx context.Context, corrID, sessionID, name string, args map[string]any) (string, error) {
	r.mu.RLock()
	t, ok := r.tools[name]
	r.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("unknown tool: %q", name)
	}
	ctx = context.WithValue(ctx, SessionKey, sessionID)
	out, err := t.Fn(ctx, args)
	if err != nil {
		if r.log != nil {
			r.log.Error("tool failed", "tool", name, "corr_id", corrID, "session_id", sessionID, "err", err)
		}
		return "", fmt.Errorf("tool %q: %w", name, err)
	}
	res := truncate.Truncate(out, truncate.DefaultOptions())
	return res.Output, nil
}
