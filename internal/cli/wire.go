package cli

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/morefun2602/opencode-go/internal/bus"
	"github.com/morefun2602/opencode-go/internal/config"
	"github.com/morefun2602/opencode-go/internal/filewatcher"
	"github.com/morefun2602/opencode-go/internal/llm"
	"github.com/morefun2602/opencode-go/internal/lsp"
	"github.com/morefun2602/opencode-go/internal/mcp"
	"github.com/morefun2602/opencode-go/internal/plugin"
	"github.com/morefun2602/opencode-go/internal/policy"
	"github.com/morefun2602/opencode-go/internal/runtime"
	"github.com/morefun2602/opencode-go/internal/skill"
	"github.com/morefun2602/opencode-go/internal/snapshot"
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

	defaultModel := cfg.Model
	if defaultModel == "" && cfg.DefaultModel != "" {
		defaultModel = cfg.DefaultModel
	}
	if defaultModel == "" && cfg.DefaultProvider != "" {
		defaultModel = cfg.DefaultProvider
	}
	router := llm.NewRouter(registry, defaultModel, cfg.SmallModel)

	var prov llm.Provider
	p, _, err := router.ResolveDefault()
	if err == nil {
		prov = p
	}
	if prov == nil {
		name := cfg.DefaultProvider
		if name == "" {
			name = cfg.LLMProvider
		}
		if name != "" {
			prov, _ = registry.Get(name)
		}
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
	skillSearchPaths := []string{}
	if cfg.WorkspaceRoot != "" {
		skillSearchPaths = append(skillSearchPaths,
			filepath.Join(cfg.WorkspaceRoot, ".cursor", "skills"),
			filepath.Join(cfg.WorkspaceRoot, ".agents", "skills"),
		)
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		skillSearchPaths = append(skillSearchPaths,
			filepath.Join(home, ".cursor", "skills"),
			filepath.Join(home, ".agents", "skills"),
		)
	}
	dir := cfg.SkillsDir
	if dir == "" {
		dir = filepath.Join(cfg.DataDir, "skills")
	}
	skillSearchPaths = append(skillSearchPaths, dir)
	skills, _ = skill.DiscoverSkills(skillSearchPaths)
	if len(skills) == 0 {
		skills, _ = skill.LoadDir(dir)
	}

	b := bus.New()

	var watcher *filewatcher.Watcher
	if cfg.WorkspaceRoot != "" {
		watcher = filewatcher.New(cfg.WorkspaceRoot, b, log)
	}

	tool.RegisterBuiltin(treg, pol, skills, watcher)

	var snap *snapshot.Service
	if cfg.WorkspaceRoot != "" {
		snap = snapshot.New(cfg.WorkspaceRoot, log)
		if !snap.Available() {
			snap = nil
		}
	}

	for _, ls := range cfg.LSP.Servers {
		client, lerr := lsp.NewClient(context.Background(), ls.Command, ls.Args, cfg.WorkspaceRoot, log)
		if lerr != nil {
			log.Warn("lsp_start_fail", "language", ls.Language, "err", lerr)
			continue
		}
		tool.RegisterLSP(treg, client)
	}

	eng := &runtime.Engine{
		Store:              st,
		LLM:                prov,
		Router:             router,
		Providers:          registry,
		Tools:              route,
		Policy:             pol,
		Log:                log,
		Bus:                b,
		Skills:             skills,
		Agent:              runtime.AgentBuild,
		MaxToolRounds:      cfg.MaxToolRounds,
		LLMMaxRetries:      cfg.LLMMaxRetries,
		CompactionTurns:    cfg.CompactionTurns,
		WorkspaceRoot:      cfg.WorkspaceRoot,
		ConfigInstructions: cfg.Instructions,
		CompactionConfig:   cfg.Compaction,
		Compaction:         tools.NewCompactor(),
	}

	if snap != nil {
		eng.Snapshot = snap
	}

	tool.RegisterTask(treg, eng, st, cfg.WorkspaceID, 2)

	return eng, st, nil
}
