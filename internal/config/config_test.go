package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParse(t *testing.T) {
	tests := []struct {
		name      string
		content   string // empty means don't create file
		create    bool
		wantErr   bool
		wantCount int // expected number of servers
	}{
		{
			name: "valid YAML",
			content: `servers:
  myserver:
    command: npx
    args: ["-y", "some-mcp"]
    env:
      TOKEN: abc
    daemon: true
    startup_timeout: 5s
`,
			create:    true,
			wantCount: 1,
		},
		{
			name:      "empty file",
			content:   "",
			create:    true,
			wantCount: 0,
		},
		{
			name:      "missing file returns empty",
			create:    false,
			wantCount: 0,
		},
		{
			name:    "malformed YAML",
			content: "servers:\n  - this is not valid:\n\t\tbad",
			create:  true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yml")

			if tt.create {
				writeFile(t, dir, "config.yml", tt.content)
			}

			cfg, err := parse(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := len(cfg.Servers)
			if got != tt.wantCount {
				t.Errorf("server count = %d, want %d", got, tt.wantCount)
			}
		})
	}
}

func TestParse_DefaultTransport(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yml", `servers:
  s1:
    command: foo
`)
	cfg, err := parse(filepath.Join(dir, "config.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Servers["s1"].Transport != "stdio" {
		t.Errorf("transport = %q, want %q", cfg.Servers["s1"].Transport, "stdio")
	}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name    string
		global  *Config
		project *Config
		want    map[string]string // server name -> expected command
	}{
		{
			name: "no overlap",
			global: &Config{Servers: map[string]*ServerConfig{
				"a": {Command: "cmd-a", Transport: "stdio"},
			}},
			project: &Config{Servers: map[string]*ServerConfig{
				"b": {Command: "cmd-b", Transport: "stdio"},
			}},
			want: map[string]string{"a": "cmd-a", "b": "cmd-b"},
		},
		{
			name: "overlap project wins",
			global: &Config{Servers: map[string]*ServerConfig{
				"a": {Command: "global-a", Transport: "stdio"},
			}},
			project: &Config{Servers: map[string]*ServerConfig{
				"a": {Command: "project-a", Transport: "stdio"},
			}},
			want: map[string]string{"a": "project-a"},
		},
		{
			name:   "empty global",
			global: &Config{},
			project: &Config{Servers: map[string]*ServerConfig{
				"b": {Command: "cmd-b", Transport: "stdio"},
			}},
			want: map[string]string{"b": "cmd-b"},
		},
		{
			name: "empty project",
			global: &Config{Servers: map[string]*ServerConfig{
				"a": {Command: "cmd-a", Transport: "stdio"},
			}},
			project: &Config{},
			want:    map[string]string{"a": "cmd-a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged := Merge(tt.global, tt.project)
			if len(merged.Servers) != len(tt.want) {
				t.Fatalf("server count = %d, want %d", len(merged.Servers), len(tt.want))
			}
			for name, wantCmd := range tt.want {
				sc, ok := merged.Servers[name]
				if !ok {
					t.Errorf("missing server %q", name)
					continue
				}
				if sc.Command != wantCmd {
					t.Errorf("server %q command = %q, want %q", name, sc.Command, wantCmd)
				}
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid stdio",
			cfg: &Config{Servers: map[string]*ServerConfig{
				"s": {Command: "npx", Transport: "stdio"},
			}},
		},
		{
			name: "valid sse",
			cfg: &Config{Servers: map[string]*ServerConfig{
				"s": {Transport: "sse", URL: "http://localhost:8080"},
			}},
		},
		{
			name: "valid http",
			cfg: &Config{Servers: map[string]*ServerConfig{
				"s": {Transport: "http", URL: "https://mcp.example.com"},
			}},
		},
		{
			name: "missing url for http",
			cfg: &Config{Servers: map[string]*ServerConfig{
				"s": {Transport: "http"},
			}},
			wantErr: true,
		},
		{
			name: "missing command for stdio",
			cfg: &Config{Servers: map[string]*ServerConfig{
				"s": {Transport: "stdio"},
			}},
			wantErr: true,
		},
		{
			name: "missing url for sse",
			cfg: &Config{Servers: map[string]*ServerConfig{
				"s": {Transport: "sse"},
			}},
			wantErr: true,
		},
		{
			name: "unknown transport",
			cfg: &Config{Servers: map[string]*ServerConfig{
				"s": {Command: "x", Transport: "grpc"},
			}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.cfg)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestLoadFrom(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()

	globalPath := writeFile(t, globalDir, "config.yml", `servers:
  shared:
    command: global-shared
  global-only:
    command: global-cmd
`)

	projectPath := writeFile(t, projectDir, "config.yml", `servers:
  shared:
    command: project-shared
  project-only:
    command: project-cmd
`)

	cfg, err := LoadFrom(globalPath, projectPath)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	// Should have 3 servers: shared (project wins), global-only, project-only.
	if len(cfg.Servers) != 3 {
		t.Fatalf("server count = %d, want 3", len(cfg.Servers))
	}

	if cfg.Servers["shared"].Command != "project-shared" {
		t.Errorf("shared command = %q, want %q", cfg.Servers["shared"].Command, "project-shared")
	}
	if cfg.Servers["global-only"].Command != "global-cmd" {
		t.Errorf("global-only command = %q, want %q", cfg.Servers["global-only"].Command, "global-cmd")
	}
	if cfg.Servers["project-only"].Command != "project-cmd" {
		t.Errorf("project-only command = %q, want %q", cfg.Servers["project-only"].Command, "project-cmd")
	}
}

func TestLoadFrom_MissingFiles(t *testing.T) {
	cfg, err := LoadFrom("/nonexistent/global.yml", "/nonexistent/project.yml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Servers) != 0 {
		t.Errorf("expected 0 servers, got %d", len(cfg.Servers))
	}
}

func TestLoadFrom_ValidationError(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config.yml", `servers:
  bad:
    transport: stdio
`)

	_, err := LoadFrom(path, "")
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}
