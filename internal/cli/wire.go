package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/morefun2602/opencode-go/internal/bus"
	"github.com/morefun2602/opencode-go/internal/config"
	"github.com/morefun2602/opencode-go/internal/filewatcher"
	"github.com/morefun2602/opencode-go/internal/llm"
	"github.com/morefun2602/opencode-go/internal/lsp"
	"github.com/morefun2602/opencode-go/internal/mcp"
	"github.com/morefun2602/opencode-go/internal/permission"
	"github.com/morefun2602/opencode-go/internal/plugin"
	"github.com/morefun2602/opencode-go/internal/policy"
	"github.com/morefun2602/opencode-go/internal/providerstate"
	"github.com/morefun2602/opencode-go/internal/runtime"
	"github.com/morefun2602/opencode-go/internal/skill"
	"github.com/morefun2602/opencode-go/internal/snapshot"
	"github.com/morefun2602/opencode-go/internal/store"
	"github.com/morefun2602/opencode-go/internal/tool"
	"github.com/morefun2602/opencode-go/internal/tools"
)

// BuildSkillSearchPaths constructs the ordered list of directories to search for skills.
// Exported so the CLI skills list command can use the same logic.
func BuildSkillSearchPaths(cfg config.Config, log *slog.Logger) []string {
	var paths []string

	disableExternal := os.Getenv("OPENCODE_DISABLE_EXTERNAL_SKILLS") != ""

	if !disableExternal {
		if cfg.WorkspaceRoot != "" {
			paths = append(paths,
				filepath.Join(cfg.WorkspaceRoot, ".cursor", "skills"),
				filepath.Join(cfg.WorkspaceRoot, ".agents", "skills"),
			)
		}
		home, _ := os.UserHomeDir()
		if home != "" {
			paths = append(paths,
				filepath.Join(home, ".cursor", "skills"),
				filepath.Join(home, ".agents", "skills"),
			)
		}
	}

	for _, p := range cfg.Skills.Paths {
		expanded := p
		if strings.HasPrefix(expanded, "~/") {
			home, _ := os.UserHomeDir()
			if home != "" {
				expanded = filepath.Join(home, expanded[2:])
			}
		}
		if !filepath.IsAbs(expanded) {
			expanded = filepath.Join(cfg.WorkspaceRoot, expanded)
		}
		if info, err := os.Stat(expanded); err != nil || !info.IsDir() {
			log.Warn("skill path not found", "path", expanded)
			continue
		}
		paths = append(paths, expanded)
	}

	for _, u := range cfg.Skills.URLs {
		disc := skill.NewDiscovery(filepath.Join(cfg.DataDir, "cache"), log)
		dirs := disc.Pull(u)
		paths = append(paths, dirs...)
	}

	dir := cfg.SkillsDir
	if dir == "" {
		dir = filepath.Join(cfg.DataDir, "skills")
	}
	paths = append(paths, dir)

	return paths
}

func wireEngine(cfg config.Config, log *slog.Logger) (*runtime.Engine, store.Store, error) {
	if err := plugin.StartAll(context.Background()); err != nil {
		return nil, nil, err
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("create data dir %s: %w", cfg.DataDir, err)
	}
	path := filepath.Join(cfg.DataDir, "sqlite.db")
	st, err := store.Open(path)
	if err != nil {
		return nil, nil, err
	}

	ps, _ := providerstate.Build(context.Background(), cfg, providerstate.BuildOptions{
		DisableModelsFetch: os.Getenv("OPENCODE_DISABLE_MODELS_FETCH") != "",
		ModelsURL:          os.Getenv("OPENCODE_MODELS_URL"),
	})

	registry := llm.NewRegistry()
	for pname, ap := range ps.Providers {
		n, a := pname, ap
		model := ""
		if len(a.Models) > 0 {
			model = a.Models[0]
		}
		p := llm.NewProvider(n, llm.ProviderConfig{
			APIKey:  a.APIKey,
			BaseURL: a.BaseURL,
			Model:   model,
			Models:  a.Models,
			Type:    a.Type,
		})
		registry.Register(n, func() llm.Provider { return p })
	}

	defaultModel := cfg.Model
	if defaultModel == "" {
		defaultModel = ps.Default
	}
	if defaultModel == "" && cfg.DefaultModel != "" {
		defaultModel = cfg.DefaultModel
	}
	if defaultModel == "" && cfg.DefaultProvider != "" {
		defaultModel = cfg.DefaultProvider
	}
	smallModel := cfg.SmallModel
	if smallModel == "" {
		smallModel = ps.Small
	}
	router := llm.NewRouter(registry, defaultModel, smallModel)

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
		srv := s
		transportType := srv.InferTransport()
		var inner *mcpclient.Client
		var cerr error
		timeout := 30 * time.Second
		if srv.TimeoutSec > 0 {
			timeout = time.Duration(srv.TimeoutSec) * time.Second
		}

		var oauthProvider *mcp.OAuthProvider
		headerFunc := func(ctx context.Context) map[string]string {
			headers := map[string]string{}
			for k, v := range srv.Headers {
				headers[k] = v
			}
			if oauthProvider != nil {
				token, err := oauthProvider.GetToken(ctx)
				if err != nil {
					log.Warn("mcp_oauth_header_failed", "server", srv.Name, "err", err)
				} else if token != nil && token.AccessToken != "" {
					tokenType := token.TokenType
					if tokenType == "" {
						tokenType = "Bearer"
					}
					headers["Authorization"] = tokenType + " " + token.AccessToken
				}
			}
			return headers
		}

		if srv.OAuth != nil {
			oauthProvider = mcp.NewOAuthProvider(srv.Name, mcp.OAuthConfig{
				AuthorizationURL: srv.OAuth.AuthorizationURL,
				TokenURL:         srv.OAuth.TokenURL,
				ClientID:         srv.OAuth.ClientID,
				ClientSecret:     srv.OAuth.ClientSecret,
				Scopes:           srv.OAuth.Scopes,
				RedirectPort:     srv.OAuth.RedirectPort,
			}, log)
		}

		switch transportType {
		case "stdio":
			inner, cerr = mcpclient.NewStdioMCPClient(srv.Command, nil, srv.Args...)
		case "sse":
			opts := []transport.ClientOption{}
			if len(srv.Headers) > 0 || oauthProvider != nil {
				opts = append(opts, transport.WithHeaderFunc(headerFunc))
			}
			inner, cerr = mcpclient.NewSSEMCPClient(srv.URL, opts...)
		case "streamable_http":
			opts := []transport.StreamableHTTPCOption{}
			if len(srv.Headers) > 0 || oauthProvider != nil {
				opts = append(opts, transport.WithHTTPHeaderFunc(headerFunc))
			}
			if timeout > 0 {
				opts = append(opts, transport.WithHTTPTimeout(timeout))
			}
			inner, cerr = mcpclient.NewStreamableHttpClient(srv.URL, opts...)
		default:
			log.Warn("mcp_unknown_transport", "server", srv.Name, "transport", transportType)
			continue
		}
		if cerr != nil {
			log.Warn("mcp_start_fail", "server", srv.Name, "err", cerr)
			continue
		}
		c, cerr := mcp.NewClient(inner, srv.Name, cfg.MCPToolPrefix, log, oauthProvider, timeout)
		if cerr != nil {
			log.Warn("mcp_connect_fail", "server", srv.Name, "err", cerr)
			continue
		}
		mcpClients = append(mcpClients, c)
	}

	route := &tool.Router{Builtin: treg, Clients: mcpClients, Log: log}

	var skills []skill.Skill
	skillSearchPaths := BuildSkillSearchPaths(cfg, log)
	skills, _ = skill.DiscoverSkills(skillSearchPaths, log)

	b := bus.New()

	var watcher *filewatcher.Watcher
	if cfg.WorkspaceRoot != "" {
		watcher = filewatcher.New(cfg.WorkspaceRoot, b, log)
	}

	tool.RegisterBuiltin(treg, pol, skills, watcher)
	tool.RegisterCustomToolsFromWorkspace(treg, cfg.WorkspaceRoot, log)

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

	registerCustomAgents(cfg, log)

	agentSwitch := runtime.NewAgentSwitch()

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
		AgentSwitch:        agentSwitch,
		MaxToolRounds:      cfg.MaxToolRounds,
		DoomLoopWindow:     cfg.DoomLoopWindow,
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

	planSwitch := &planSwitchAdapter{as: agentSwitch}
	tool.RegisterPlan(treg, planSwitch)

	tool.RegisterTask(treg, eng, st, cfg.WorkspaceID, 2,
		func() []tool.SubagentInfo {
			subs := runtime.ListSubagents()
			out := make([]tool.SubagentInfo, len(subs))
			for i, s := range subs {
				out[i] = tool.SubagentInfo{Name: s.Name, Description: s.Description, CanUse: true}
			}
			return out
		},
		func(name string) (tool.SubagentInfo, error) {
			a, ok := runtime.GetAgent(name)
			if !ok {
				subs := runtime.ListSubagents()
				names := make([]string, len(subs))
				for i, s := range subs {
					names[i] = s.Name
				}
				return tool.SubagentInfo{}, fmt.Errorf("unknown agent %q, available: %s", name, strings.Join(names, ", "))
			}
			return tool.SubagentInfo{
				Name:        a.Name,
				Description: a.Description,
				CanUse:      !a.Hidden && a.Subagent,
			}, nil
		},
	)

	return eng, st, nil
}

// planSwitchAdapter adapts runtime.AgentSwitch to the tool.PlanSwitch interface.
type planSwitchAdapter struct {
	as *runtime.AgentSwitch
}

func (p *planSwitchAdapter) IsInPlan(sessionID string) bool {
	a, ok := p.as.Get(sessionID)
	return ok && a.Name == "plan"
}

func (p *planSwitchAdapter) EnterPlan(sessionID string) {
	p.as.Set(sessionID, runtime.AgentPlan)
}

func (p *planSwitchAdapter) ExitPlan(sessionID string) {
	p.as.Delete(sessionID)
}

func registerCustomAgents(cfg config.Config, log *slog.Logger) {
	for _, af := range cfg.Agents {
		if af.Name == "" {
			continue
		}
		a := runtime.Agent{
			Name:        af.Name,
			Description: af.Description,
			Prompt:      af.Prompt,
			Model:       af.Model,
			Steps:       af.Steps,
			Hidden:      af.Hidden,
			Subagent:    af.Subagent,
		}

		mode := runtime.ModeBuild
		if af.Mode != "" {
			if m, ok := runtime.GetMode(af.Mode); ok {
				mode = m
			}
		}
		a.Mode = mode

		if af.Temp > 0 {
			t := af.Temp
			a.Temperature = &t
		}

		if len(af.Tools) > 0 {
			var rules permission.Ruleset
			rules = append(rules, permission.Rule{Permission: "*", Pattern: "*", Action: permission.ActionDeny})
			for _, t := range af.Tools {
				rules = append(rules, permission.Rule{Permission: t, Pattern: "*", Action: permission.ActionAllow})
			}
			a.Permission = rules
		}

		runtime.RegisterAgent(a)
		log.Info("registered_custom_agent", "name", af.Name)
	}
}
