package cli

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/morefun2602/opencode-go/internal/bus"
	"github.com/morefun2602/opencode-go/internal/config"
	"github.com/morefun2602/opencode-go/internal/llm"
	"github.com/morefun2602/opencode-go/internal/mcp"
	"github.com/morefun2602/opencode-go/internal/plugin"
	"github.com/morefun2602/opencode-go/internal/policy"
	"github.com/morefun2602/opencode-go/internal/runtime"
	"github.com/morefun2602/opencode-go/internal/skill"
	"github.com/morefun2602/opencode-go/internal/store"
	"github.com/morefun2602/opencode-go/internal/tool"
	"github.com/morefun2602/opencode-go/internal/tools"
)

func wireEngine(cfg config.Config, log *slog.Logger) (*runtime.Engine, store.Store, error) {
	if err := plugin.StartAll(context.Background()); err != nil {
		return nil, nil, err
	}
	path := filepath.Join(cfg.DataDir, "sqlite.db")
	st, err := store.Open(path)
	if err != nil {
		return nil, nil, err
	}

	registry := llm.NewRegistry()
	for pname, pf := range cfg.Providers {
		n, f := pname, pf
		p := llm.NewProvider(n, llm.ProviderConfig{
			APIKey:  f.APIKey,
			BaseURL: f.BaseURL,
			Model:   f.Model,
			Type:    f.Type,
		})
		registry.Register(n, func() llm.Provider { return p })
	}

	name := cfg.DefaultProvider
	if name == "" {
		name = cfg.LLMProvider
	}
	var prov llm.Provider
	if name != "" {
		prov, _ = registry.Get(name)
	}
	if prov == nil {
		prov = llm.Stub{}
	}

	pol := &policy.Policy{
		WorkspaceRoot:       cfg.WorkspaceRoot,
		RequireWriteConfirm: cfg.RequireWriteConfirm,
		BashTimeoutSec:      cfg.BashTimeoutSec,
		MaxOutputBytes:      cfg.MaxOutputBytes,
		Permissions:         cfg.Permissions,
		Log:                 log,
	}
	treg := tools.New(log)
	tool.RegisterBuiltin(treg, pol)

	var mcpClients []*mcp.Client
	for _, s := range cfg.MCPServers {
		transport := s.InferTransport()
		var inner *mcpclient.Client
		var cerr error
		switch transport {
		case "stdio":
			inner, cerr = mcpclient.NewStdioMCPClient(s.Command, nil, s.Args...)
		case "sse":
			inner, cerr = mcpclient.NewSSEMCPClient(s.URL)
		case "streamable_http":
			inner, cerr = mcpclient.NewStreamableHttpClient(s.URL)
		default:
			log.Warn("mcp_unknown_transport", "server", s.Name, "transport", transport)
			continue
		}
		if cerr != nil {
			log.Warn("mcp_start_fail", "server", s.Name, "err", cerr)
			continue
		}
		c, cerr := mcp.NewClient(inner, s.Name, cfg.MCPToolPrefix, log)
		if cerr != nil {
			log.Warn("mcp_connect_fail", "server", s.Name, "err", cerr)
			continue
		}
		mcpClients = append(mcpClients, c)
	}

	route := &tool.Router{Builtin: treg, Clients: mcpClients, Log: log}

	var skills []skill.Skill
	dir := cfg.SkillsDir
	if dir == "" {
		dir = filepath.Join(cfg.DataDir, "skills")
	}
	skills, _ = skill.LoadDir(dir)

	var sysPrompt string
	if len(cfg.Instructions) > 0 {
		sysPrompt = strings.Join(cfg.Instructions, "\n") + "\n\n"
	}

	b := bus.New()
	eng := &runtime.Engine{
		Store:           st,
		LLM:             prov,
		Providers:       registry,
		Tools:           route,
		Policy:          pol,
		Log:             log,
		Bus:             b,
		Skills:          skills,
		MaxToolRounds:   cfg.MaxToolRounds,
		LLMMaxRetries:   cfg.LLMMaxRetries,
		CompactionTurns: cfg.CompactionTurns,
		SystemPrompt:    sysPrompt,
	}

	tool.RegisterTask(treg, eng, cfg.WorkspaceID, 2)

	return eng, st, nil
}
