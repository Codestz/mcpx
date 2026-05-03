package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the full mcpx configuration.
type Config struct {
	Servers  map[string]*ServerConfig `yaml:"servers"`
	Security *SecurityConfig          `yaml:"security"`
	Gain     *GainConfig              `yaml:"gain"`
	Cache    *CacheConfig             `yaml:"cache"`
	UI       *UIConfig                `yaml:"ui"`
}

// ServerConfig describes a single MCP server.
type ServerConfig struct {
	Command        string            `yaml:"command"`
	Args           []string          `yaml:"args"`
	Transport      string            `yaml:"transport"`
	Env            map[string]string `yaml:"env"`
	Daemon         bool              `yaml:"daemon"`
	StartupTimeout string            `yaml:"startup_timeout"`
	URL            string            `yaml:"url"`
	Headers        map[string]string `yaml:"headers"`
	Auth           *AuthConfig       `yaml:"auth"`
	Security       *ServerSecurity   `yaml:"security"`
}

// AuthConfig holds authentication settings for remote transports.
type AuthConfig struct {
	Type  string `yaml:"type"`
	Token string `yaml:"token"`
}

// SecurityConfig holds the top-level security configuration.
type SecurityConfig struct {
	Enabled bool           `yaml:"enabled"`
	Global  GlobalSecurity `yaml:"global"`
}

// GlobalSecurity holds security settings that apply to all servers.
type GlobalSecurity struct {
	Audit     *AuditConfig     `yaml:"audit"`
	RateLimit *RateLimitConfig `yaml:"rate_limit"`
	Policies  []Policy         `yaml:"policies"`
}

// AuditConfig controls audit logging.
type AuditConfig struct {
	Enabled bool     `yaml:"enabled"`
	Log     string   `yaml:"log"`
	Redact  []string `yaml:"redact"`
}

// RateLimitConfig controls per-server rate limiting.
type RateLimitConfig struct {
	MaxCallsPerMinute int `yaml:"max_calls_per_minute"`
	MaxCallsPerTool   int `yaml:"max_calls_per_tool"`
}

// Policy defines a security rule evaluated before tool calls.
type Policy struct {
	Name    string      `yaml:"name"`
	Match   PolicyMatch `yaml:"match"`
	Action  string      `yaml:"action"`  // "allow", "deny", "warn"
	Message string      `yaml:"message"`
}

// PolicyMatch defines what a policy matches against.
type PolicyMatch struct {
	Tools   []string            `yaml:"tools"`
	Args    map[string]ArgRule  `yaml:"args"`
	Content *ContentMatch       `yaml:"content"`
}

// ArgRule defines rules for matching argument values.
type ArgRule struct {
	DenyPattern string   `yaml:"deny_pattern"`
	AllowPrefix []string `yaml:"allow_prefix"`
	DenyPrefix  []string `yaml:"deny_prefix"`
}

// ContentMatch inspects the body/value of a specific argument.
type ContentMatch struct {
	Target         string `yaml:"target"`          // dot-path to arg, e.g. "args.sql"
	DenyPattern    string `yaml:"deny_pattern"`
	RequirePattern string `yaml:"require_pattern"`
	When           string `yaml:"when"`            // only apply require_pattern when this matches
}

// ServerSecurity holds per-server security overrides.
type ServerSecurity struct {
	Mode         string   `yaml:"mode"`          // "read-only", "editing", "custom"
	AllowedTools []string `yaml:"allowed_tools"`
	BlockedTools []string `yaml:"blocked_tools"`
	Policies     []Policy `yaml:"policies"`
}

// GainConfig controls token-savings analytics and JSONL stats.
type GainConfig struct {
	Enabled     *bool  `yaml:"enabled"`
	Tokenizer   string `yaml:"tokenizer"`     // "estimate" (default) | "tiktoken"
	RetainDays  int    `yaml:"retain_days"`   // 0 = no retention pruning
	StatsPath   string `yaml:"stats_path"`    // override default ~/.mcpx/stats.jsonl
}

// CacheConfig controls schema and result caches.
type CacheConfig struct {
	SchemaTTL          string   `yaml:"schema_ttl"`            // duration, e.g. "5m"
	ResultTTL          string   `yaml:"result_ttl"`            // duration, e.g. "30s"
	ResultEnabled      *bool    `yaml:"result_enabled"`
	ResultHeuristic    bool     `yaml:"result_heuristic"`      // opt-in get_/list_/find_/...
	ResultIdempotent   []string `yaml:"result_idempotent_tools"` // "server.tool" entries
}

// UIConfig controls the always-on dashboard.
type UIConfig struct {
	Enabled     *bool  `yaml:"enabled"`
	Port        int    `yaml:"port"`         // 0 = ephemeral
	Bind        string `yaml:"bind"`         // default "127.0.0.1"
	IdleTimeout string `yaml:"idle_timeout"` // duration, default "1h"
}


// Default returns the GainConfig with defaults applied.
func (g *GainConfig) Default() GainConfig {
	out := GainConfig{Tokenizer: "estimate", RetainDays: 30}
	if g != nil {
		if g.Enabled != nil {
			out.Enabled = g.Enabled
		}
		if g.Tokenizer != "" {
			out.Tokenizer = g.Tokenizer
		}
		if g.RetainDays != 0 {
			out.RetainDays = g.RetainDays
		}
		out.StatsPath = g.StatsPath
	}
	if out.Enabled == nil {
		t := true
		out.Enabled = &t
	}
	return out
}

// Default returns the CacheConfig with defaults applied.
func (c *CacheConfig) Default() CacheConfig {
	out := CacheConfig{SchemaTTL: "5m", ResultTTL: "30s"}
	if c != nil {
		if c.SchemaTTL != "" {
			out.SchemaTTL = c.SchemaTTL
		}
		if c.ResultTTL != "" {
			out.ResultTTL = c.ResultTTL
		}
		out.ResultEnabled = c.ResultEnabled
		out.ResultHeuristic = c.ResultHeuristic
		out.ResultIdempotent = c.ResultIdempotent
	}
	if out.ResultEnabled == nil {
		t := true
		out.ResultEnabled = &t
	}
	return out
}

// Default returns the UIConfig with defaults applied.
func (u *UIConfig) Default() UIConfig {
	out := UIConfig{Port: 7878, Bind: "127.0.0.1", IdleTimeout: "1h"}
	if u != nil {
		if u.Enabled != nil {
			out.Enabled = u.Enabled
		}
		if u.Port != 0 {
			out.Port = u.Port
		}
		if u.Bind != "" {
			out.Bind = u.Bind
		}
		if u.IdleTimeout != "" {
			out.IdleTimeout = u.IdleTimeout
		}
	}
	if out.Enabled == nil {
		t := true
		out.Enabled = &t
	}
	return out
}

// Load reads the global (~/.mcpx/config.yml) and project (.mcpx/config.yml)
// configs, merges them, and validates the result.
// Returns the merged config, the project root (empty if no project config), and any error.
func Load() (*Config, string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, "", fmt.Errorf("config: user home dir: %w", err)
	}
	globalPath := filepath.Join(home, ".mcpx", "config.yml")

	global, err := parse(globalPath)
	if err != nil {
		return nil, "", fmt.Errorf("config: global config: %w", err)
	}

	projectPath, projectRoot, err := findProjectConfig()
	if err != nil {
		return nil, "", fmt.Errorf("config: find project config: %w", err)
	}

	project, err := parse(projectPath)
	if err != nil {
		return nil, "", fmt.Errorf("config: project config: %w", err)
	}

	merged := Merge(global, project)
	if err := Validate(merged); err != nil {
		return nil, "", err
	}

	return merged, projectRoot, nil
}

// LoadFrom loads configs from explicit paths (useful for testing).
func LoadFrom(globalPath, projectPath string) (*Config, error) {
	global, err := parse(globalPath)
	if err != nil {
		return nil, fmt.Errorf("config: global config: %w", err)
	}

	project, err := parse(projectPath)
	if err != nil {
		return nil, fmt.Errorf("config: project config: %w", err)
	}

	merged := Merge(global, project)
	if err := Validate(merged); err != nil {
		return nil, err
	}

	return merged, nil
}

// parse reads and unmarshals a single YAML config file.
// Returns an empty Config if the file does not exist.
func parse(path string) (*Config, error) {
	if path == "" {
		return &Config{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}

	// Apply defaults.
	for _, sc := range cfg.Servers {
		if sc.Transport == "" {
			sc.Transport = "stdio"
		}
	}

	return &cfg, nil
}

// Merge combines global and project configs. Project servers replace global
// servers entirely on a per-server-name basis. Project security overrides global.
func Merge(global, project *Config) *Config {
	merged := &Config{
		Servers: make(map[string]*ServerConfig),
	}

	for name, sc := range global.Servers {
		merged.Servers[name] = sc
	}
	for name, sc := range project.Servers {
		merged.Servers[name] = sc
	}

	// Security: project wins if present, otherwise global.
	merged.Security = global.Security
	if project.Security != nil {
		merged.Security = project.Security
	}

	// Gain/Cache/UI: project overrides global at the section level.
	merged.Gain = global.Gain
	if project.Gain != nil {
		merged.Gain = project.Gain
	}
	merged.Cache = global.Cache
	if project.Cache != nil {
		merged.Cache = project.Cache
	}
	merged.UI = global.UI
	if project.UI != nil {
		merged.UI = project.UI
	}

	return merged
}

// Validate checks that every server in the config has the required fields
// for its transport type.
func Validate(cfg *Config) error {
	for name, sc := range cfg.Servers {
		switch sc.Transport {
		case "stdio", "":
			if sc.Command == "" {
				return fmt.Errorf("config: server %q: command is required for stdio transport", name)
			}
		case "http":
			if sc.URL == "" {
				return fmt.Errorf("config: server %q: url is required for http transport", name)
			}
		case "sse":
			if sc.URL == "" {
				return fmt.Errorf("config: server %q: url is required for sse transport", name)
			}
		default:
			return fmt.Errorf("config: server %q: unknown transport %q", name, sc.Transport)
		}

		// Validate security mode.
		if sc.Security != nil && sc.Security.Mode != "" {
			switch sc.Security.Mode {
			case "read-only", "editing", "custom":
			default:
				return fmt.Errorf("config: server %q: unknown security mode %q (must be read-only, editing, or custom)", name, sc.Security.Mode)
			}
		}

		// Validate policy actions.
		if sc.Security != nil {
			for _, p := range sc.Security.Policies {
				if err := validatePolicyAction(name, p); err != nil {
					return err
				}
			}
		}
	}

	// Validate global policies.
	if cfg.Security != nil {
		for _, p := range cfg.Security.Global.Policies {
			if err := validatePolicyAction("(global)", p); err != nil {
				return err
			}
		}
	}

	return nil
}

func validatePolicyAction(serverName string, p Policy) error {
	switch p.Action {
	case "allow", "deny", "warn":
		return nil
	case "":
		return fmt.Errorf("config: server %q: policy %q missing action", serverName, p.Name)
	default:
		return fmt.Errorf("config: server %q: policy %q has unknown action %q (must be allow, deny, or warn)", serverName, p.Name, p.Action)
	}
}

// findProjectConfig walks up from the current working directory looking for
// .mcpx/config.yml. Returns (configPath, projectRoot, error).
// Returns empty strings if no project config is found.
func findProjectConfig() (string, string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("config: getwd: %w", err)
	}

	for {
		candidate := filepath.Join(dir, ".mcpx", "config.yml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root.
			return "", "", nil
		}
		dir = parent
	}
}
