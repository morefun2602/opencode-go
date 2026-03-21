package tools

import (
	"context"
	"fmt"
	"sync"
)

type pendingQ struct {
	text    string
	options []string
	ch      chan string
}

type QuestionManager struct {
	mu      sync.Mutex
	pending map[string]*pendingQ
}

var Questions = &QuestionManager{pending: map[string]*pendingQ{}}

func (q *QuestionManager) Ask(ctx context.Context, id, text string, options []string) (string, error) {
	pq := &pendingQ{text: text, options: options, ch: make(chan string, 1)}
	q.mu.Lock()
	q.pending[id] = pq
	q.mu.Unlock()
	defer func() {
		q.mu.Lock()
		delete(q.pending, id)
		q.mu.Unlock()
	}()
	select {
	case ans := <-pq.ch:
		return ans, nil
	case <-ctx.Done():
		return "", fmt.Errorf("question %q: %w", id, ctx.Err())
	}
}

func (q *QuestionManager) Reply(id, answer string) bool {
	q.mu.Lock()
	pq, ok := q.pending[id]
	q.mu.Unlock()
	if !ok {
		return false
	}
	pq.ch <- answer
	return true
}

type PendingQuestionInfo struct {
	ID      string
	Text    string
	Options []string
}

func (q *QuestionManager) List() []PendingQuestionInfo {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]PendingQuestionInfo, 0, len(q.pending))
	for id, pq := range q.pending {
		out = append(out, PendingQuestionInfo{ID: id, Text: pq.text, Options: pq.options})
	}
	return out
}
