package runtime

import (
	"sync"
)

// Manager 进程内会话 ID 唯一性（与 store 协同；store 为跨进程真相源）。
type Manager struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func NewManager() *Manager {
	return &Manager{seen: map[string]struct{}{}}
}

func (m *Manager) Track(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.seen[id]; ok {
		return false
	}
	m.seen[id] = struct{}{}
	return true
}

func (m *Manager) Release(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.seen, id)
}
