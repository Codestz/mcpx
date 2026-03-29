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

func TestParse_SecurityConfig(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config.yml", `security:
  enabled: true
  global:
    audit:
      enabled: true
      log: /tmp/audit.jsonl
      redact: ["$(secret.*)"]
    rate_limit:
      max_calls_per_minute: 60
    policies:
      - name: no-traversal
        match:
          args:
            path:
              deny_pattern: "\\.\\.\\/|\\.\\.\\\\\\/"
        action: deny
        message: "Path traversal blocked"

servers:
  myserver:
    command: echo
    security:
      mode: read-only
      allowed_tools: [search, list]
      blocked_tools: [delete]
      policies:
        - name: restrict
          match:
            tools: ["query"]
            content:
              target: args.sql
              deny_pattern: "(?i)DROP"
          action: deny
          message: "DROP blocked"
    lifecycle:
      on_connect:
        - tool: activate_project
          args:
            project: /tmp/myproject
`)

	cfg, err := parse(path)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// Top-level security.
	if cfg.Security == nil {
		t.Fatal("expected security config, got nil")
	}
	if !cfg.Security.Enabled {
		t.Error("security.enabled = false, want true")
	}
	if cfg.Security.Global.Audit == nil {
		t.Fatal("expected audit config")
	}
	if cfg.Security.Global.Audit.Log != "/tmp/audit.jsonl" {
		t.Errorf("audit.log = %q, want /tmp/audit.jsonl", cfg.Security.Global.Audit.Log)
	}
	if cfg.Security.Global.RateLimit == nil || cfg.Security.Global.RateLimit.MaxCallsPerMinute != 60 {
		t.Errorf("rate_limit.max_calls_per_minute != 60")
	}
	if len(cfg.Security.Global.Policies) != 1 {
		t.Fatalf("global policies count = %d, want 1", len(cfg.Security.Global.Policies))
	}
	if cfg.Security.Global.Policies[0].Action != "deny" {
		t.Errorf("global policy action = %q, want deny", cfg.Security.Global.Policies[0].Action)
	}

	// Per-server security.
	sc := cfg.Servers["myserver"]
	if sc.Security == nil {
		t.Fatal("expected server security config")
	}
	if sc.Security.Mode != "read-only" {
		t.Errorf("security.mode = %q, want read-only", sc.Security.Mode)
	}
	if len(sc.Security.AllowedTools) != 2 {
		t.Errorf("allowed_tools count = %d, want 2", len(sc.Security.AllowedTools))
	}
	if len(sc.Security.BlockedTools) != 1 {
		t.Errorf("blocked_tools count = %d, want 1", len(sc.Security.BlockedTools))
	}
	if len(sc.Security.Policies) != 1 {
		t.Fatalf("server policies count = %d, want 1", len(sc.Security.Policies))
	}
	if sc.Security.Policies[0].Match.Content == nil {
		t.Fatal("expected content match in server policy")
	}
	if sc.Security.Policies[0].Match.Content.Target != "args.sql" {
		t.Errorf("content.target = %q, want args.sql", sc.Security.Policies[0].Match.Content.Target)
	}

	// Lifecycle.
	if sc.Lifecycle == nil {
		t.Fatal("expected lifecycle config")
	}
	if len(sc.Lifecycle.OnConnect) != 1 {
		t.Fatalf("on_connect hooks count = %d, want 1", len(sc.Lifecycle.OnConnect))
	}
	hook := sc.Lifecycle.OnConnect[0]
	if hook.Tool != "activate_project" {
		t.Errorf("hook.tool = %q, want activate_project", hook.Tool)
	}
	if hook.Args["project"] != "/tmp/myproject" {
		t.Errorf("hook.args.project = %v, want /tmp/myproject", hook.Args["project"])
	}
}

func TestValidate_SecurityMode(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		wantErr bool
	}{
		{"read-only", "read-only", false},
		{"editing", "editing", false},
		{"custom", "custom", false},
		{"empty is ok", "", false},
		{"invalid", "yolo", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Servers: map[string]*ServerConfig{
				"s": {Command: "x", Transport: "stdio", Security: &ServerSecurity{Mode: tt.mode}},
			}}
			err := Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_PolicyAction(t *testing.T) {
	tests := []struct {
		name    string
		action  string
		wantErr bool
	}{
		{"allow", "allow", false},
		{"deny", "deny", false},
		{"warn", "warn", false},
		{"empty", "", true},
		{"invalid", "block", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Servers: map[string]*ServerConfig{
				"s": {
					Command:   "x",
					Transport: "stdio",
					Security: &ServerSecurity{
						Policies: []Policy{{Name: "test", Action: tt.action}},
					},
				},
			}}
			err := Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_LifecycleHookMissingTool(t *testing.T) {
	cfg := &Config{Servers: map[string]*ServerConfig{
		"s": {
			Command:   "x",
			Transport: "stdio",
			Lifecycle: &LifecycleConfig{
				OnConnect: []LifecycleHook{{Tool: ""}},
			},
		},
	}}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for empty hook tool name")
	}
}

func TestParse_Workspaces(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config.yml", `servers:
  serena:
    command: serena
    workspaces:
      - name: frontend
        path: packages/web
        lifecycle:
          on_connect:
            - tool: activate_project
              args:
                project: /monorepo/packages/web
        security:
          mode: read-only
      - name: backend
        path: services/api
        lifecycle:
          on_connect:
            - tool: activate_project
              args:
                project: /monorepo/services/api
        security:
          mode: editing
`)

	cfg, err := parse(path)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	sc := cfg.Servers["serena"]
	if len(sc.Workspaces) != 2 {
		t.Fatalf("workspaces count = %d, want 2", len(sc.Workspaces))
	}

	ws0 := sc.Workspaces[0]
	if ws0.Name != "frontend" {
		t.Errorf("ws[0].name = %q, want frontend", ws0.Name)
	}
	if ws0.Path != "packages/web" {
		t.Errorf("ws[0].path = %q, want packages/web", ws0.Path)
	}
	if ws0.Security == nil || ws0.Security.Mode != "read-only" {
		t.Errorf("ws[0].security.mode != read-only")
	}
	if ws0.Lifecycle == nil || len(ws0.Lifecycle.OnConnect) != 1 {
		t.Fatal("ws[0] missing lifecycle hooks")
	}

	ws1 := sc.Workspaces[1]
	if ws1.Name != "backend" {
		t.Errorf("ws[1].name = %q, want backend", ws1.Name)
	}
	if ws1.Security == nil || ws1.Security.Mode != "editing" {
		t.Errorf("ws[1].security.mode != editing")
	}
}

func TestResolveWorkspace(t *testing.T) {
	// Create a temp directory to simulate project root with workspaces.
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "packages", "web", "src"), 0o755)
	os.MkdirAll(filepath.Join(root, "services", "api", "internal"), 0o755)
	os.MkdirAll(filepath.Join(root, "docs"), 0o755)

	sc := &ServerConfig{
		Command:   "serena",
		Transport: "stdio",
		Workspaces: []WorkspaceConfig{
			{Name: "frontend", Path: "packages/web"},
			{Name: "backend", Path: "services/api"},
		},
	}

	tests := []struct {
		name    string
		cwd     string
		wantWS  string
		wantNil bool
	}{
		{"inside frontend", filepath.Join(root, "packages", "web", "src"), "frontend", false},
		{"at frontend root", filepath.Join(root, "packages", "web"), "frontend", false},
		{"inside backend", filepath.Join(root, "services", "api", "internal"), "backend", false},
		{"at project root", root, "", true},
		{"in docs (no workspace)", filepath.Join(root, "docs"), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Change cwd for test.
			origDir, _ := os.Getwd()
			os.Chdir(tt.cwd)
			defer os.Chdir(origDir)

			ws := ResolveWorkspace(sc, root)
			if tt.wantNil {
				if ws != nil {
					t.Errorf("expected nil, got workspace %q", ws.Name)
				}
				return
			}
			if ws == nil {
				t.Fatalf("expected workspace %q, got nil", tt.wantWS)
			}
			if ws.Name != tt.wantWS {
				t.Errorf("workspace = %q, want %q", ws.Name, tt.wantWS)
			}
		})
	}
}

func TestValidate_WorkspaceMissingName(t *testing.T) {
	cfg := &Config{Servers: map[string]*ServerConfig{
		"s": {
			Command:   "x",
			Transport: "stdio",
			Workspaces: []WorkspaceConfig{
				{Path: "packages/web"},
			},
		},
	}}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for workspace missing name")
	}
}

func TestValidate_WorkspaceMissingPath(t *testing.T) {
	cfg := &Config{Servers: map[string]*ServerConfig{
		"s": {
			Command:   "x",
			Transport: "stdio",
			Workspaces: []WorkspaceConfig{
				{Name: "frontend"},
			},
		},
	}}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for workspace missing path")
	}
}

func TestValidate_GlobalPolicyAction(t *testing.T) {
	cfg := &Config{
		Servers: map[string]*ServerConfig{
			"s": {Command: "x", Transport: "stdio"},
		},
		Security: &SecurityConfig{
			Global: GlobalSecurity{
				Policies: []Policy{{Name: "bad", Action: "nope"}},
			},
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for invalid global policy action")
	}
}
