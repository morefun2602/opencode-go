package llm

import (
	"fmt"
	"sync"
)

// Registry 注册 LLM 提供商名称 -> 构造器。
type Registry struct {
	mu   sync.RWMutex
	fact map[string]func() Provider
}

func NewRegistry() *Registry {
	return &Registry{fact: map[string]func() Provider{}}
}

func (r *Registry) Register(name string, fn func() Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fact[name] = fn
}

func (r *Registry) Get(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fn, ok := r.fact[name]
	if !ok {
		return nil, fmt.Errorf("unknown llm provider: %q", name)
	}
	return fn(), nil
}

func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.fact))
	for n := range r.fact {
		names = append(names, n)
	}
	return names
}
