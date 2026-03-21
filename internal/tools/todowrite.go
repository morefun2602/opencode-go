package tools

import (
	"fmt"
	"strings"
	"sync"
)

type sessionKeyType struct{}

var SessionKey = sessionKeyType{}

type TodoItem struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Status  string `json:"status"`
}

type TodoStore struct {
	mu    sync.RWMutex
	items map[string][]TodoItem
}

var GlobalTodos = &TodoStore{items: map[string][]TodoItem{}}

func (s *TodoStore) Set(session string, todos []TodoItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[session] = todos
}

func (s *TodoStore) Merge(session string, todos []TodoItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing := s.items[session]
	idx := map[string]int{}
	for i, t := range existing {
		idx[t.ID] = i
	}
	for _, t := range todos {
		if i, ok := idx[t.ID]; ok {
			if t.Content != "" {
				existing[i].Content = t.Content
			}
			if t.Status != "" {
				existing[i].Status = t.Status
			}
		} else {
			existing = append(existing, t)
		}
	}
	s.items[session] = existing
}

func (s *TodoStore) Get(session string) []TodoItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]TodoItem{}, s.items[session]...)
}

func FormatTodos(session string) string {
	items := GlobalTodos.Get(session)
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## Current TODOs\n")
	for _, t := range items {
		marker := "[ ]"
		switch t.Status {
		case "completed":
			marker = "[x]"
		case "in_progress":
			marker = "[~]"
		}
		fmt.Fprintf(&b, "- %s %s: %s\n", marker, t.ID, t.Content)
	}
	return b.String()
}
