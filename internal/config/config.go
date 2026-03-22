package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tailscale/hujson"
)

const DefaultUpstreamCompatRef = "0.0.0-placeholder-sync-with-upstream"

// ProviderFile matches the original opencode provider config format.
// JSON key in opencode.json: "provider": { "<name>": { "options": { ... } } }
type ProviderFile struct {
	Options struct {
		APIKey  string `json:"apiKey"`
		BaseURL string `json:"baseURL"`
	} `json:"options"`
}

// InternalProvider is the runtime representation used by the engine.
type InternalProvider struct {
	APIKey  string
	BaseURL string
	Model   string
	Type    string
}

// AgentFile matches the original opencode agent config format.
// JSON key in opencode.json: "agent": { "<name>": { ... } }
type AgentFile struct {
	Name        string   `json:"-"`
	Description string   `json:"description"`
	Tools       []string `json:"tools"`
	Model       string   `json:"model"`
	Temp        float64  `json:"temperature"`
	Steps       int      `json:"steps"`
	Prompt      string   `json:"prompt"`
	Mode        string   `json:"mode"`
	Hidden      bool     `json:"hidden"`
	Subagent    bool     `json:"subagent"`
}

// MCPFile matches the original opencode MCP server config format.
// JSON key in opencode.json: "mcp": { "<name>": { ... } }
type MCPFile struct {
	Type        string            `json:"type"`
	Command     []string          `json:"command"`
	URL         string            `json:"url"`
	Enabled     *bool             `json:"enabled"`
	Environment map[string]string `json:"environment"`
}

// MCPServerFile is the runtime representation used by the engine.
type MCPServerFile struct {
	Name      string   `json:"name"`
	Command   string   `json:"command"`
	Args      []string `json:"args,omitempty"`
	Transport string   `json:"transport"`
	URL       string   `json:"url,omitempty"`
}

// InferTransport returns the effective transport for this MCP server config.
func (m MCPServerFile) InferTransport() string {
	if m.Transport != "" {
		return m.Transport
	}
	if m.Command != "" {
		return "stdio"
	}
	if m.URL != "" {
		return "streamable_http"
	}
	return "stdio"
}

// File is the JSON on-disk representation, fully compatible with original opencode.
type File struct {
	Schema            string `json:"$schema,omitempty"`
	UpstreamCompatRef string `json:"upstream_compat_ref,omitempty"`

	Server struct {
		Listen    string `json:"listen"`
		AuthToken string `json:"auth_token"`
	} `json:"server"`

	// Original opencode top-level fields
	Provider     map[string]ProviderFile `json:"provider"`
	Model        string                  `json:"model"`
	SmallModel   string                  `json:"small_model"`
	MCP          map[string]MCPFile      `json:"mcp"`
	Agent        map[string]AgentFile    `json:"agent"`
	Permission   map[string]string       `json:"permission"`
	Instructions []string                `json:"instructions"`
	Skills       SkillsConfig            `json:"skills"`
	Compaction   CompactionConfig        `json:"compaction"`
	Lsp          map[string]LSPFile      `json:"lsp"`

	// Go extension fields (optional, kept at top level)
	DataDir                string `json:"data_dir,omitempty"`
	WorkspaceID            string `json:"workspace_id,omitempty"`
	WorkspaceRoot          string `json:"workspace_root,omitempty"`
	BashTimeoutSec         int    `json:"bash_timeout_sec,omitempty"`
	MaxOutputBytes         int    `json:"max_output_bytes,omitempty"`
	MaxToolRounds          int    `json:"max_tool_rounds,omitempty"`
	LLMMaxRetries          int    `json:"llm_max_retries,omitempty"`
	LLMTimeout             string `json:"llm_timeout,omitempty"`
	MCPToolPrefix          string `json:"mcp_tool_prefix,omitempty"`
	RemoteConfigURL        string `json:"remote_config_url,omitempty"`
	SkillsDir              string `json:"skills_dir,omitempty"`
	CompactionTurns        int    `json:"compaction_turns,omitempty"`
	RequireWriteConfirm    bool   `json:"require_write_confirm,omitempty"`
	StructuredOutputSchema string `json:"structured_output_schema,omitempty"`
}

// LSPFile matches the original opencode LSP config format.
// JSON key: "lsp": { "<language>": { "command": ["cmd", "args..."] } } or false
type LSPFile struct {
	Command []string `json:"command"`
}

type CompactionConfig struct {
	Auto     *bool `json:"auto"`
	Reserved int   `json:"reserved"`
	Prune    *bool `json:"prune"`
}

func (c CompactionConfig) AutoEnabled() bool {
	if c.Auto == nil {
		return true
	}
	return *c.Auto
}

func (c CompactionConfig) PruneEnabled() bool {
	if c.Prune == nil {
		return true
	}
	return *c.Prune
}

func (c CompactionConfig) ReservedTokens() int {
	if c.Reserved > 0 {
		return c.Reserved
	}
	return 20000
}

type LSPServer struct {
	Language string
	Command  string
	Args     []string
}

type LSPConfig struct {
	Servers []LSPServer
}

type SkillsConfig struct {
	Paths []string `json:"paths"`
	URLs  []string `json:"urls"`
}

// Config is the runtime configuration consumed by the engine.
type Config struct {
	UpstreamCompatRef      string
	Listen                 string
	AuthToken              string
	WorkspaceID            string
	DataDir                string
	LLMTimeout             time.Duration
	ConfigPath             string
	LLMProvider            string
	WorkspaceRoot          string
	RequireWriteConfirm    bool
	BashTimeoutSec         int
	MaxOutputBytes         int
	CompactionTurns        int
	LLMMaxRetries          int
	StructuredOutputSchema string
	MCPServers             []MCPServerFile
	MCPToolPrefix          string
	Providers              map[string]InternalProvider
	DefaultProvider        string
	DefaultModel           string
	MaxToolRounds          int
	Permissions            map[string]string
	SkillsDir              string
	Agents                 []AgentFile
	Instructions           []string
	RemoteConfigURL        string
	Model                  string
	SmallModel             string
	Compaction             CompactionConfig
	LSP                    LSPConfig
	Skills                 SkillsConfig
}

func Defaults() Config {
	return Config{
		UpstreamCompatRef: DefaultUpstreamCompatRef,
		Listen:            "127.0.0.1:8080",
		WorkspaceID:       "default",
		DataDir:           ".opencode",
		LLMTimeout:        60 * time.Second,
		WorkspaceRoot:     ".",
		BashTimeoutSec:    30,
		MaxOutputBytes:    256 * 1024,
		MCPToolPrefix:     "mcp.",
		MaxToolRounds:     25,
	}
}

func (c *Config) Validate() error {
	if !isLoopbackOnly(c.Listen) && strings.TrimSpace(c.AuthToken) == "" {
		return fmt.Errorf("非 loopback 监听时必须配置 server.auth_token（或 OPENCODE_AUTH_TOKEN）")
	}
	return nil
}

func isLoopbackOnly(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		if strings.HasPrefix(addr, ":") {
			return true
		}
		h := strings.Trim(addr, "[]")
		return h == "127.0.0.1" || h == "::1" || h == "localhost"
	}
	h := strings.Trim(host, "[]")
	if h == "0.0.0.0" || h == "::" {
		return false
	}
	return h == "127.0.0.1" || h == "::1" || h == "localhost"
}

// merge applies a parsed File onto the runtime Config.
func merge(dst *Config, src File) {
	if src.UpstreamCompatRef != "" {
		dst.UpstreamCompatRef = src.UpstreamCompatRef
	}
	if src.Server.Listen != "" {
		dst.Listen = src.Server.Listen
	}
	if src.Server.AuthToken != "" {
		dst.AuthToken = src.Server.AuthToken
	}

	// Provider: convert original opencode format to internal
	if len(src.Provider) > 0 {
		if dst.Providers == nil {
			dst.Providers = make(map[string]InternalProvider)
		}
		for name, pf := range src.Provider {
			p := dst.Providers[name]
			if pf.Options.APIKey != "" {
				p.APIKey = pf.Options.APIKey
			}
			if pf.Options.BaseURL != "" {
				p.BaseURL = pf.Options.BaseURL
			}
			if p.Type == "" {
				p.Type = name
			}
			dst.Providers[name] = p
		}
	}

	if src.Model != "" {
		dst.Model = src.Model
	}
	if src.SmallModel != "" {
		dst.SmallModel = src.SmallModel
	}

	// MCP: convert map[name]MCPFile to []MCPServerFile
	if len(src.MCP) > 0 {
		dst.MCPServers = nil
		for name, mf := range src.MCP {
			if mf.Enabled != nil && !*mf.Enabled {
				continue
			}
			s := MCPServerFile{Name: name}
			if mf.URL != "" {
				s.URL = mf.URL
				s.Transport = "streamable_http"
				if mf.Type == "remote" {
					s.Transport = "streamable_http"
				}
			}
			if len(mf.Command) > 0 {
				s.Command = mf.Command[0]
				if len(mf.Command) > 1 {
					s.Args = mf.Command[1:]
				}
				s.Transport = "stdio"
			}
			if mf.Type != "" && s.Transport == "" {
				switch mf.Type {
				case "local":
					s.Transport = "stdio"
				case "remote":
					s.Transport = "streamable_http"
				}
			}
			dst.MCPServers = append(dst.MCPServers, s)
		}
	}

	// Agent: convert map[name]AgentFile to []AgentFile
	if len(src.Agent) > 0 {
		dst.Agents = nil
		for name, af := range src.Agent {
			af.Name = name
			dst.Agents = append(dst.Agents, af)
		}
	}

	if len(src.Permission) > 0 {
		dst.Permissions = make(map[string]string, len(src.Permission))
		for k, v := range src.Permission {
			dst.Permissions[k] = v
		}
	}

	if len(src.Instructions) > 0 {
		dst.Instructions = append([]string(nil), src.Instructions...)
	}

	if src.Compaction.Auto != nil || src.Compaction.Reserved > 0 || src.Compaction.Prune != nil {
		dst.Compaction = src.Compaction
	}

	// LSP: convert map[language]LSPFile to LSPConfig
	if len(src.Lsp) > 0 {
		dst.LSP.Servers = nil
		for lang, lf := range src.Lsp {
			if len(lf.Command) == 0 {
				continue
			}
			s := LSPServer{Language: lang, Command: lf.Command[0]}
			if len(lf.Command) > 1 {
				s.Args = lf.Command[1:]
			}
			dst.LSP.Servers = append(dst.LSP.Servers, s)
		}
	}

	if len(src.Skills.Paths) > 0 {
		dst.Skills.Paths = append([]string(nil), src.Skills.Paths...)
	}
	if len(src.Skills.URLs) > 0 {
		dst.Skills.URLs = append([]string(nil), src.Skills.URLs...)
	}

	// Go extension fields
	if src.DataDir != "" {
		dst.DataDir = src.DataDir
	}
	if src.WorkspaceID != "" {
		dst.WorkspaceID = src.WorkspaceID
	}
	if src.WorkspaceRoot != "" {
		dst.WorkspaceRoot = src.WorkspaceRoot
	}
	if src.LLMTimeout != "" {
		if d, err := time.ParseDuration(src.LLMTimeout); err == nil {
			dst.LLMTimeout = d
		}
	}
	if src.BashTimeoutSec > 0 {
		dst.BashTimeoutSec = src.BashTimeoutSec
	}
	if src.MaxOutputBytes > 0 {
		dst.MaxOutputBytes = src.MaxOutputBytes
	}
	if src.MaxToolRounds > 0 {
		dst.MaxToolRounds = src.MaxToolRounds
	}
	if src.LLMMaxRetries > 0 {
		dst.LLMMaxRetries = src.LLMMaxRetries
	}
	if src.CompactionTurns > 0 {
		dst.CompactionTurns = src.CompactionTurns
	}
	if src.MCPToolPrefix != "" {
		dst.MCPToolPrefix = src.MCPToolPrefix
	}
	if src.RemoteConfigURL != "" {
		dst.RemoteConfigURL = src.RemoteConfigURL
	}
	dst.RequireWriteConfirm = src.RequireWriteConfirm
	if src.StructuredOutputSchema != "" {
		dst.StructuredOutputSchema = src.StructuredOutputSchema
	}
	if src.SkillsDir != "" {
		dst.SkillsDir = src.SkillsDir
	}
}

func applyEnv(c *Config) {
	if v := os.Getenv("OPENCODE_UPSTREAM_COMPAT_REF"); v != "" {
		c.UpstreamCompatRef = v
	}
	if v := os.Getenv("OPENCODE_SERVER_LISTEN"); v != "" {
		c.Listen = v
	}
	if v := os.Getenv("OPENCODE_AUTH_TOKEN"); v != "" {
		c.AuthToken = v
	}
	if v := os.Getenv("OPENCODE_WORKSPACE_ID"); v != "" {
		c.WorkspaceID = v
	}
	if v := os.Getenv("OPENCODE_DATA_DIR"); v != "" {
		c.DataDir = v
	}
	if v := os.Getenv("OPENCODE_LLM_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.LLMTimeout = d
		}
	}
	if v := os.Getenv("OPENCODE_BASH_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.BashTimeoutSec = n
		}
	}
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		if c.Providers == nil {
			c.Providers = map[string]InternalProvider{}
		}
		p := c.Providers["openai"]
		if p.APIKey == "" {
			p.APIKey = v
			if p.Type == "" {
				p.Type = "openai"
			}
			c.Providers["openai"] = p
		}
	}
	if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
		if c.Providers == nil {
			c.Providers = map[string]InternalProvider{}
		}
		p := c.Providers["anthropic"]
		if p.APIKey == "" {
			p.APIKey = v
			if p.Type == "" {
				p.Type = "anthropic"
			}
			c.Providers["anthropic"] = p
		}
	}
	if v := os.Getenv("OPENCODE_API_KEY"); v != "" {
		if c.Providers == nil {
			c.Providers = map[string]InternalProvider{}
		}
		p := c.Providers["opencode"]
		if p.APIKey == "" {
			p.APIKey = v
			if p.Type == "" {
				p.Type = "opencode"
			}
			c.Providers["opencode"] = p
		}
	}
}

// loadFile reads and parses a JSON/JSONC config file at path.
func loadFile(path string) (File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return File{}, err
	}
	standardized, err := hujson.Standardize(b)
	if err != nil {
		return File{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	var f File
	if err := json.Unmarshal(standardized, &f); err != nil {
		return File{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	return f, nil
}

// Load reads configuration from multiple config files (lowest to highest priority),
// then overlays environment variables and CLI flags.
//
// Search order (later overrides earlier):
//  1. ~/.config/opencode/opencode.json (user global)
//  2. .opencode/opencode.jsonc (workspace)
//  3. opencode.json (CWD)
//  4. $OPENCODE_CONFIG or explicit configPath (highest)
func Load(configPath string, flags *Config) (Config, error) {
	c := Defaults()

	// 1. User global config
	if home, err := os.UserHomeDir(); err == nil {
		for _, name := range []string{"opencode.json", "opencode.jsonc"} {
			p := filepath.Join(home, ".config", "opencode", name)
			if f, err := loadFile(p); err == nil {
				merge(&c, f)
				if c.ConfigPath == "" {
					c.ConfigPath = p
				}
				break
			}
		}
	}

	// 2. Workspace .opencode/ directory
	for _, name := range []string{"opencode.jsonc", "opencode.json"} {
		p := filepath.Join(".opencode", name)
		if f, err := loadFile(p); err == nil {
			merge(&c, f)
			c.ConfigPath = p
			break
		}
	}

	// 3. CWD opencode.json
	if _, err := os.Stat("opencode.json"); err == nil {
		if f, err := loadFile("opencode.json"); err == nil {
			merge(&c, f)
			c.ConfigPath = "opencode.json"
		}
	}

	// 4. Explicit path or $OPENCODE_CONFIG
	explicit := configPath
	if explicit == "" {
		explicit = os.Getenv("OPENCODE_CONFIG")
	}
	if explicit != "" {
		f, err := loadFile(explicit)
		if err != nil && !os.IsNotExist(err) {
			return c, err
		}
		if err == nil {
			merge(&c, f)
			c.ConfigPath = explicit
		}
	}

	// Remote config fallback
	if c.RemoteConfigURL != "" {
		if rf, err := fetchRemote(c.RemoteConfigURL); err == nil {
			mergeRemoteFallback(&c, rf)
		}
	}

	applyEnv(&c)

	if flags != nil {
		mergeFlags(&c, flags)
	}

	return c, c.Validate()
}

func fetchRemote(url string) (File, error) {
	if !strings.HasPrefix(url, "https://") {
		return File{}, fmt.Errorf("remote config must use HTTPS")
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return File{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return File{}, err
	}
	var f File
	if err := json.Unmarshal(body, &f); err != nil {
		return File{}, err
	}
	return f, nil
}

func mergeRemoteFallback(dst *Config, remote File) {
	if dst.Model == "" && remote.Model != "" {
		dst.Model = remote.Model
	}
	if dst.SmallModel == "" && remote.SmallModel != "" {
		dst.SmallModel = remote.SmallModel
	}
	if len(dst.Providers) == 0 && len(remote.Provider) > 0 {
		dst.Providers = make(map[string]InternalProvider, len(remote.Provider))
		for name, pf := range remote.Provider {
			dst.Providers[name] = InternalProvider{
				APIKey:  pf.Options.APIKey,
				BaseURL: pf.Options.BaseURL,
				Type:    name,
			}
		}
	}
	if len(dst.MCPServers) == 0 && len(remote.MCP) > 0 {
		for name, mf := range remote.MCP {
			if mf.Enabled != nil && !*mf.Enabled {
				continue
			}
			s := MCPServerFile{Name: name}
			if len(mf.Command) > 0 {
				s.Command = mf.Command[0]
				if len(mf.Command) > 1 {
					s.Args = mf.Command[1:]
				}
				s.Transport = "stdio"
			}
			if mf.URL != "" {
				s.URL = mf.URL
				s.Transport = "streamable_http"
			}
			dst.MCPServers = append(dst.MCPServers, s)
		}
	}
	if len(dst.Permissions) == 0 && len(remote.Permission) > 0 {
		dst.Permissions = make(map[string]string, len(remote.Permission))
		for k, v := range remote.Permission {
			dst.Permissions[k] = v
		}
	}
	if len(dst.Agents) == 0 && len(remote.Agent) > 0 {
		for name, af := range remote.Agent {
			af.Name = name
			dst.Agents = append(dst.Agents, af)
		}
	}
	if len(dst.Instructions) == 0 && len(remote.Instructions) > 0 {
		dst.Instructions = append([]string(nil), remote.Instructions...)
	}
	if dst.StructuredOutputSchema == "" && remote.StructuredOutputSchema != "" {
		dst.StructuredOutputSchema = remote.StructuredOutputSchema
	}
}

func mergeFlags(dst *Config, flags *Config) {
	if flags.Listen != "" {
		dst.Listen = flags.Listen
	}
	if flags.AuthToken != "" {
		dst.AuthToken = flags.AuthToken
	}
	if flags.WorkspaceID != "" {
		dst.WorkspaceID = flags.WorkspaceID
	}
	if flags.DataDir != "" {
		dst.DataDir = flags.DataDir
	}
	if flags.LLMTimeout > 0 {
		dst.LLMTimeout = flags.LLMTimeout
	}
	if flags.ConfigPath != "" {
		dst.ConfigPath = flags.ConfigPath
	}
	if flags.LLMProvider != "" {
		dst.LLMProvider = flags.LLMProvider
	}
	if flags.WorkspaceRoot != "" {
		dst.WorkspaceRoot = flags.WorkspaceRoot
	}
	if flags.RequireWriteConfirm {
		dst.RequireWriteConfirm = true
	}
	if flags.BashTimeoutSec > 0 {
		dst.BashTimeoutSec = flags.BashTimeoutSec
	}
	if flags.MaxOutputBytes > 0 {
		dst.MaxOutputBytes = flags.MaxOutputBytes
	}
	if flags.DefaultProvider != "" {
		dst.DefaultProvider = flags.DefaultProvider
	}
	if flags.MaxToolRounds > 0 {
		dst.MaxToolRounds = flags.MaxToolRounds
	}
	if flags.Model != "" {
		dst.Model = flags.Model
	}
	if flags.SmallModel != "" {
		dst.SmallModel = flags.SmallModel
	}
}
