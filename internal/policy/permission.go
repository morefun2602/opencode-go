package policy

import (
	"context"
	"fmt"
	"sync"
)

type PermissionReply struct {
	Action string // "allow" | "deny"
	Scope  string // "once" | "always" | "reject"
}

type PermissionManager struct {
	mu      sync.Mutex
	pending map[string]chan PermissionReply
}

var Permissions = &PermissionManager{pending: make(map[string]chan PermissionReply)}

func (m *PermissionManager) Ask(ctx context.Context, id, tool, arg string) (PermissionReply, error) {
	m.mu.Lock()
	ch := make(chan PermissionReply, 1)
	m.pending[id] = ch
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		delete(m.pending, id)
		m.mu.Unlock()
	}()

	select {
	case <-ctx.Done():
		return PermissionReply{}, fmt.Errorf("permission request %s cancelled: %w", id, ctx.Err())
	case r := <-ch:
		return r, nil
	}
}

func (m *PermissionManager) Reply(id string, r PermissionReply) bool {
	m.mu.Lock()
	ch, ok := m.pending[id]
	m.mu.Unlock()
	if !ok {
		return false
	}
	select {
	case ch <- r:
		return true
	default:
		return false
	}
}
