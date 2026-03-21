package policy

import (
	"log/slog"
	"path"
	"strings"
	"sync"
)

type Policy struct {
	WorkspaceRoot       string
	RequireWriteConfirm bool
	BashTimeoutSec      int
	MaxOutputBytes      int
	SearchURL           string
	Permissions         map[string]string // tool name or "tool:argPattern" -> "allow" | "ask" | "deny"
	Cache               *PermissionCache
	Log                 *slog.Logger
}

func (p *Policy) CheckPermission(name string) string {
	return p.CheckPermissionWithArg(name, "")
}

func (p *Policy) CheckPermissionWithArg(name, arg string) string {
	if p == nil || p.Permissions == nil {
		return "allow"
	}
	if p.Cache != nil {
		if v, ok := p.Cache.Check(name, arg); ok {
			return v
		}
	}
	if v, ok := p.Permissions[name]; ok {
		return v
	}
	for pattern, action := range p.Permissions {
		parts := strings.SplitN(pattern, ":", 2)
		if len(parts) == 1 {
			if matched, _ := path.Match(parts[0], name); matched {
				return action
			}
			continue
		}
		toolPat, argPat := parts[0], parts[1]
		toolMatch, _ := path.Match(toolPat, name)
		if !toolMatch {
			continue
		}
		if arg != "" {
			if argMatch, _ := path.Match(argPat, arg); argMatch {
				return action
			}
		}
	}
	return "allow"
}

func (p *Policy) RecordDecision(tool, arg, scope string) {
	if p == nil || p.Cache == nil {
		return
	}
	switch scope {
	case "always":
		p.Cache.Set(tool, arg, "allow")
	case "reject":
		p.Cache.Set(tool, arg, "deny")
	}
}

func (p *Policy) Audit(msg string, kv ...any) {
	if p == nil || p.Log == nil {
		return
	}
	p.Log.Info(msg, kv...)
}

// PermissionCache stores session-level permission decisions.
type PermissionCache struct {
	mu    sync.RWMutex
	rules map[string]string // "tool\x00arg" -> "allow" | "deny"
}

func NewPermissionCache() *PermissionCache {
	return &PermissionCache{rules: make(map[string]string)}
}

func cacheKey(tool, arg string) string {
	return tool + "\x00" + arg
}

func (c *PermissionCache) Set(tool, arg, action string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rules[cacheKey(tool, arg)] = action
}

func (c *PermissionCache) Check(tool, arg string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.rules[cacheKey(tool, arg)]
	return v, ok
}
