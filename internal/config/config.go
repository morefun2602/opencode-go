package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/tailscale/hujson"
)

const DefaultUpstreamCompatRef = "0.0.0-placeholder-sync-with-upstream"

type ProviderFile struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
	Type    string `json:"type"`
}

type AgentFile struct {
	Name        string   `json:"name"`
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

type MCPServerFile struct {
	Name      string   `json:"name"`
	Command   string   `json:"command"`
	Args      []string `json:"args"`
	Transport string   `json:"transport"` // "stdio" | "sse" | "streamable_http"
	URL       string   `json:"url"`
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

type File struct {
	UpstreamCompatRef string `json:"upstream_compat_ref"`
	Server            struct {
		Listen    string `json:"listen"`
		AuthToken string `json:"auth_token"`
	} `json:"server"`
	Workspace struct {
		ID string `json:"id"`
	} `json:"workspace"`
	Go struct {
		DataDir                string                  `json:"data_dir"`
		LLMTimeout             string                  `json:"llm_timeout"`
		LLMProvider            string                  `json:"llm_provider"`
		WorkspaceRoot          string                  `json:"workspace_root"`
		RequireWriteConfirm    bool                    `json:"require_write_confirm"`
		BashTimeoutSec         int                     `json:"bash_timeout_sec"`
		MaxOutputBytes         int                     `json:"max_output_bytes"`
		CompactionTurns        int                     `json:"compaction_turns"`
		LLMMaxRetries          int                     `json:"llm_max_retries"`
		StructuredOutputSchema string                  `json:"structured_output_schema"`
		MCPServers             []MCPServerFile         `json:"mcp_servers"`
		MCPToolPrefix          string                  `json:"mcp_tool_prefix"`
		Providers              map[string]ProviderFile `json:"providers"`
		DefaultProvider        string                  `json:"default_provider"`
		DefaultModel           string                  `json:"default_model"`
		MaxToolRounds          int                     `json:"max_tool_rounds"`
		Permissions            map[string]string       `json:"permissions"`
		SkillsDir              string                  `json:"skills_dir"`
		Agents                 []AgentFile             `json:"agents"`
		Instructions           []string                `json:"instructions"`
		RemoteConfigURL        string                  `json:"remote_config_url"`
		Model                  string                  `json:"model"`
		SmallModel             string                  `json:"small_model"`
		Compaction             CompactionConfig        `json:"compaction"`
		LSP                    LSPConfig               `json:"lsp"`
		Skills                 SkillsConfig            `json:"skills"`
	} `json:"x_opencode_go"`
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
	Language string   `json:"language"`
	Command  string   `json:"command"`
	Args     []string `json:"args"`
}

type LSPConfig struct {
	Servers []LSPServer `json:"servers"`
}

type SkillsConfig struct {
	Paths []string `json:"paths"`
	URLs  []string `json:"urls"`
}

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
	Providers              map[string]ProviderFile
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
		DataDir:           ".opencode-go",
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
	if src.Workspace.ID != "" {
		dst.WorkspaceID = src.Workspace.ID
	}
	g := src.Go
	if g.DataDir != "" {
		dst.DataDir = g.DataDir
	}
	if g.LLMTimeout != "" {
		if d, err := time.ParseDuration(g.LLMTimeout); err == nil {
			dst.LLMTimeout = d
		}
	}
	if g.LLMProvider != "" {
		dst.LLMProvider = g.LLMProvider
	}
	if g.WorkspaceRoot != "" {
		dst.WorkspaceRoot = g.WorkspaceRoot
	}
	dst.RequireWriteConfirm = g.RequireWriteConfirm
	if g.BashTimeoutSec > 0 {
		dst.BashTimeoutSec = g.BashTimeoutSec
	}
	if g.MaxOutputBytes > 0 {
		dst.MaxOutputBytes = g.MaxOutputBytes
	}
	if g.CompactionTurns > 0 {
		dst.CompactionTurns = g.CompactionTurns
	}
	if g.LLMMaxRetries > 0 {
		dst.LLMMaxRetries = g.LLMMaxRetries
	}
	if g.StructuredOutputSchema != "" {
		dst.StructuredOutputSchema = g.StructuredOutputSchema
	}
	if len(g.MCPServers) > 0 {
		dst.MCPServers = append([]MCPServerFile(nil), g.MCPServers...)
	}
	if g.MCPToolPrefix != "" {
		dst.MCPToolPrefix = g.MCPToolPrefix
	}
	if len(g.Providers) > 0 {
		dst.Providers = make(map[string]ProviderFile, len(g.Providers))
		for k, v := range g.Providers {
			dst.Providers[k] = v
		}
	}
	if g.DefaultProvider != "" {
		dst.DefaultProvider = g.DefaultProvider
	}
	if g.DefaultModel != "" {
		dst.DefaultModel = g.DefaultModel
	}
	if g.MaxToolRounds > 0 {
		dst.MaxToolRounds = g.MaxToolRounds
	}
	if len(g.Permissions) > 0 {
		dst.Permissions = make(map[string]string, len(g.Permissions))
		for k, v := range g.Permissions {
			dst.Permissions[k] = v
		}
	}
	if g.SkillsDir != "" {
		dst.SkillsDir = g.SkillsDir
	}
	if len(g.Agents) > 0 {
		dst.Agents = append([]AgentFile(nil), g.Agents...)
	}
	if len(g.Instructions) > 0 {
		dst.Instructions = append([]string(nil), g.Instructions...)
	}
	if g.RemoteConfigURL != "" {
		dst.RemoteConfigURL = g.RemoteConfigURL
	}
	if g.Model != "" {
		dst.Model = g.Model
	}
	if g.SmallModel != "" {
		dst.SmallModel = g.SmallModel
	}
	if g.Compaction.Auto != nil || g.Compaction.Reserved > 0 || g.Compaction.Prune != nil {
		dst.Compaction = g.Compaction
	}
	if len(g.LSP.Servers) > 0 {
		dst.LSP = g.LSP
	}
	if len(g.Skills.Paths) > 0 {
		dst.Skills.Paths = append([]string(nil), g.Skills.Paths...)
	}
	if len(g.Skills.URLs) > 0 {
		dst.Skills.URLs = append([]string(nil), g.Skills.URLs...)
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
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		if c.Providers == nil {
			c.Providers = map[string]ProviderFile{}
		}
		p := c.Providers["openai"]
		if p.APIKey == "" {
			p.APIKey = v
			c.Providers["openai"] = p
		}
	}
	if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
		if c.Providers == nil {
			c.Providers = map[string]ProviderFile{}
		}
		p := c.Providers["anthropic"]
		if p.APIKey == "" {
			p.APIKey = v
			c.Providers["anthropic"] = p
		}
	}
}

func Load(configPath string, flags *Config) (Config, error) {
	c := Defaults()
	path := configPath
	if path == "" {
		path = os.Getenv("OPENCODE_CONFIG")
	}
	if path == "" {
		if _, err := os.Stat("opencode.json"); err == nil {
			path = "opencode.json"
		}
	}
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return c, err
		}
		if err == nil {
			standardized, hjErr := hujson.Standardize(b)
			if hjErr != nil {
				return c, fmt.Errorf("parse config %s: %w", path, hjErr)
			}
			var f File
			if err := json.Unmarshal(standardized, &f); err != nil {
				return c, fmt.Errorf("parse config %s: %w", path, err)
			}
			merge(&c, f)
			c.ConfigPath = path
		}
	}
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
	r := remote.Go
	if dst.LLMProvider == "" && r.LLMProvider != "" {
		dst.LLMProvider = r.LLMProvider
	}
	if dst.DefaultProvider == "" && r.DefaultProvider != "" {
		dst.DefaultProvider = r.DefaultProvider
	}
	if dst.DefaultModel == "" && r.DefaultModel != "" {
		dst.DefaultModel = r.DefaultModel
	}
	if len(dst.Providers) == 0 && len(r.Providers) > 0 {
		dst.Providers = make(map[string]ProviderFile, len(r.Providers))
		for k, v := range r.Providers {
			dst.Providers[k] = v
		}
	}
	if len(dst.MCPServers) == 0 && len(r.MCPServers) > 0 {
		dst.MCPServers = append([]MCPServerFile(nil), r.MCPServers...)
	}
	if len(dst.Permissions) == 0 && len(r.Permissions) > 0 {
		dst.Permissions = make(map[string]string, len(r.Permissions))
		for k, v := range r.Permissions {
			dst.Permissions[k] = v
		}
	}
	if len(dst.Agents) == 0 && len(r.Agents) > 0 {
		dst.Agents = append([]AgentFile(nil), r.Agents...)
	}
	if len(dst.Instructions) == 0 && len(r.Instructions) > 0 {
		dst.Instructions = append([]string(nil), r.Instructions...)
	}
	if dst.SkillsDir == "" && r.SkillsDir != "" {
		dst.SkillsDir = r.SkillsDir
	}
	if dst.StructuredOutputSchema == "" && r.StructuredOutputSchema != "" {
		dst.StructuredOutputSchema = r.StructuredOutputSchema
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
}
