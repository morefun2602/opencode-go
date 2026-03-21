package runtime

import "sync"

// AgentSwitch tracks per-session agent overrides for runtime mode switching.
type AgentSwitch struct {
	mu     sync.RWMutex
	agents map[string]Agent
}

func NewAgentSwitch() *AgentSwitch {
	return &AgentSwitch{agents: make(map[string]Agent)}
}

func (as *AgentSwitch) Get(sessionID string) (Agent, bool) {
	as.mu.RLock()
	defer as.mu.RUnlock()
	a, ok := as.agents[sessionID]
	return a, ok
}

func (as *AgentSwitch) Set(sessionID string, agent Agent) {
	as.mu.Lock()
	defer as.mu.Unlock()
	as.agents[sessionID] = agent
}

func (as *AgentSwitch) Delete(sessionID string) {
	as.mu.Lock()
	defer as.mu.Unlock()
	delete(as.agents, sessionID)
}
