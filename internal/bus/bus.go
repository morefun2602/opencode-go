package bus

import "sync"

type Event struct {
	Type string
	Data any
}

type Bus struct {
	mu   sync.RWMutex
	subs map[string][]chan Event
}

func New() *Bus {
	return &Bus{subs: map[string][]chan Event{}}
}

func (b *Bus) Publish(typ string, data any) {
	b.mu.RLock()
	chs := b.subs[typ]
	wildcard := b.subs["*"]
	b.mu.RUnlock()
	evt := Event{Type: typ, Data: data}
	for _, ch := range chs {
		select {
		case ch <- evt:
		default:
		}
	}
	for _, ch := range wildcard {
		select {
		case ch <- evt:
		default:
		}
	}
}

func (b *Bus) Subscribe(typ string) <-chan Event {
	ch := make(chan Event, 64)
	b.mu.Lock()
	b.subs[typ] = append(b.subs[typ], ch)
	b.mu.Unlock()
	return ch
}

func (b *Bus) SubscribeAll(types ...string) <-chan Event {
	ch := make(chan Event, 64)
	b.mu.Lock()
	for _, t := range types {
		b.subs[t] = append(b.subs[t], ch)
	}
	b.mu.Unlock()
	return ch
}

func (b *Bus) Unsubscribe(typ string, target <-chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	subs := b.subs[typ]
	for i, s := range subs {
		if (<-chan Event)(s) == target {
			b.subs[typ] = append(subs[:i], subs[i+1:]...)
			close(s)
			return
		}
	}
}
