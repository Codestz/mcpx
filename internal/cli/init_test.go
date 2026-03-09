package cli

import (
	"testing"
)

func TestConvertMCPJSON(t *testing.T) {
	tests := []struct {
		name  string
		input mcpJSON
		want  map[string]*mcpxServerConfig
	}{
		{
			name: "stdio server with daemon default",
			input: mcpJSON{
				MCPServers: map[string]mcpServerEntry{
					"serena": {
						Type:    "stdio",
						Command: "serena",
						Args:    []string{"start-mcp-server", "--context=claude-code"},
						Env:     map[string]string{},
					},
				},
			},
			want: map[string]*mcpxServerConfig{
				"serena": {
					Command:        "serena",
					Args:           []string{"start-mcp-server", "--context=claude-code"},
					Transport:      "stdio",
					Daemon:         true,
					StartupTimeout: "30s",
				},
			},
		},
		{
			name: "server with env",
			input: mcpJSON{
				MCPServers: map[string]mcpServerEntry{
					"github": {
						Type:    "stdio",
						Command: "npx",
						Args:    []string{"-y", "@modelcontextprotocol/server-github"},
						Env:     map[string]string{"GITHUB_TOKEN": "tok123"},
					},
				},
			},
			want: map[string]*mcpxServerConfig{
				"github": {
					Command:        "npx",
					Args:           []string{"-y", "@modelcontextprotocol/server-github"},
					Transport:      "stdio",
					Env:            map[string]string{"GITHUB_TOKEN": "tok123"},
					Daemon:         true,
					StartupTimeout: "30s",
				},
			},
		},
		{
			name: "sse server",
			input: mcpJSON{
				MCPServers: map[string]mcpServerEntry{
					"remote": {
						Type: "sse",
						URL:  "http://localhost:8080/sse",
					},
				},
			},
			want: map[string]*mcpxServerConfig{
				"remote": {
					Transport: "sse",
					URL:       "http://localhost:8080/sse",
				},
			},
		},
		{
			name: "default type is stdio with daemon",
			input: mcpJSON{
				MCPServers: map[string]mcpServerEntry{
					"test": {
						Command: "echo",
						Args:    []string{"hello"},
					},
				},
			},
			want: map[string]*mcpxServerConfig{
				"test": {
					Command:        "echo",
					Args:           []string{"hello"},
					Transport:      "stdio",
					Daemon:         true,
					StartupTimeout: "30s",
				},
			},
		},
		{
			name: "multiple servers",
			input: mcpJSON{
				MCPServers: map[string]mcpServerEntry{
					"a": {Command: "cmd-a", Type: "stdio"},
					"b": {Command: "cmd-b", Type: "stdio"},
				},
			},
			want: map[string]*mcpxServerConfig{
				"a": {Command: "cmd-a", Transport: "stdio", Daemon: true, StartupTimeout: "30s"},
				"b": {Command: "cmd-b", Transport: "stdio", Daemon: true, StartupTimeout: "30s"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertMCPJSON(&tt.input)
			if len(got.Servers) != len(tt.want) {
				t.Fatalf("got %d servers, want %d", len(got.Servers), len(tt.want))
			}
			for name, wantSC := range tt.want {
				gotSC, ok := got.Servers[name]
				if !ok {
					t.Errorf("missing server %q", name)
					continue
				}
				if gotSC.Command != wantSC.Command {
					t.Errorf("server %q: command = %q, want %q", name, gotSC.Command, wantSC.Command)
				}
				if gotSC.Transport != wantSC.Transport {
					t.Errorf("server %q: transport = %q, want %q", name, gotSC.Transport, wantSC.Transport)
				}
				if gotSC.URL != wantSC.URL {
					t.Errorf("server %q: url = %q, want %q", name, gotSC.URL, wantSC.URL)
				}
				if gotSC.Daemon != wantSC.Daemon {
					t.Errorf("server %q: daemon = %v, want %v", name, gotSC.Daemon, wantSC.Daemon)
				}
				if gotSC.StartupTimeout != wantSC.StartupTimeout {
					t.Errorf("server %q: startup_timeout = %q, want %q", name, gotSC.StartupTimeout, wantSC.StartupTimeout)
				}
				if len(gotSC.Args) != len(wantSC.Args) {
					t.Errorf("server %q: got %d args, want %d", name, len(gotSC.Args), len(wantSC.Args))
				}
				if len(wantSC.Env) > 0 && len(gotSC.Env) != len(wantSC.Env) {
					t.Errorf("server %q: got %d env, want %d", name, len(gotSC.Env), len(wantSC.Env))
				}
				if len(wantSC.Env) == 0 && len(gotSC.Env) != 0 {
					t.Errorf("server %q: expected empty env, got %v", name, gotSC.Env)
				}
			}
		})
	}
}
