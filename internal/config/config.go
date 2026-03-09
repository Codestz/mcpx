package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the full mcpx configuration.
type Config struct {
	Servers map[string]*ServerConfig `yaml:"servers"`
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
// servers entirely on a per-server-name basis.
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
		case "sse":
			if sc.URL == "" {
				return fmt.Errorf("config: server %q: url is required for sse transport", name)
			}
		default:
			return fmt.Errorf("config: server %q: unknown transport %q", name, sc.Transport)
		}
	}
	return nil
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
