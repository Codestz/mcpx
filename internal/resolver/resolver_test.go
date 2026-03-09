package resolver

import (
	"fmt"
	"os"
	"runtime"
	"testing"
)

// mockResolver is a test NamespaceResolver backed by a map.
type mockResolver struct {
	values map[string]string
}

func (m *mockResolver) Resolve(key string) (string, error) {
	v, ok := m.values[key]
	if !ok {
		return "", fmt.Errorf("mock: unknown key %q", key)
	}
	return v, nil
}

// mockSecretStore implements secret.Store for testing.
type mockSecretStore struct {
	data map[string]string
}

func (m *mockSecretStore) Get(name string) (string, error) {
	v, ok := m.data[name]
	if !ok {
		return "", fmt.Errorf("secret %q not found", name)
	}
	return v, nil
}

func (m *mockSecretStore) Set(name, value string) error {
	m.data[name] = value
	return nil
}

func (m *mockSecretStore) Delete(name string) error {
	delete(m.data, name)
	return nil
}

func (m *mockSecretStore) List() ([]string, error) {
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys, nil
}

func TestResolve(t *testing.T) {
	mock := &mockResolver{values: map[string]string{
		"name":    "test-project",
		"version": "1.0.0",
	}}
	r := NewWithResolvers(map[string]NamespaceResolver{
		"app": mock,
	})

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "single var",
			input: "$(app.name)",
			want:  "test-project",
		},
		{
			name:  "multiple vars",
			input: "$(app.name)-$(app.version)",
			want:  "test-project-1.0.0",
		},
		{
			name:  "no vars passthrough",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:    "unknown namespace",
			input:   "$(unknown.key)",
			wantErr: true,
		},
		{
			name:  "mixed text and vars",
			input: "name=$(app.name) ver=$(app.version) end",
			want:  "name=test-project ver=1.0.0 end",
		},
		{
			name:    "unknown key in known namespace",
			input:   "$(app.missing)",
			wantErr: true,
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.Resolve(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Resolve(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Resolve(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveSlice(t *testing.T) {
	mock := &mockResolver{values: map[string]string{"x": "42"}}
	r := NewWithResolvers(map[string]NamespaceResolver{"ns": mock})

	input := []string{"$(ns.x)", "plain", "val=$(ns.x)"}
	got, err := r.ResolveSlice(input)
	if err != nil {
		t.Fatalf("ResolveSlice() error = %v", err)
	}
	want := []string{"42", "plain", "val=42"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ResolveSlice()[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	// Error propagation.
	input = []string{"$(bad.key)"}
	_, err = r.ResolveSlice(input)
	if err == nil {
		t.Error("ResolveSlice() expected error for unknown namespace")
	}
}

func TestResolveMap(t *testing.T) {
	mock := &mockResolver{values: map[string]string{"y": "hello"}}
	r := NewWithResolvers(map[string]NamespaceResolver{"ns": mock})

	input := map[string]string{"key1": "$(ns.y)", "key2": "plain"}
	got, err := r.ResolveMap(input)
	if err != nil {
		t.Fatalf("ResolveMap() error = %v", err)
	}
	if got["key1"] != "hello" {
		t.Errorf("ResolveMap()[key1] = %q, want %q", got["key1"], "hello")
	}
	if got["key2"] != "plain" {
		t.Errorf("ResolveMap()[key2] = %q, want %q", got["key2"], "plain")
	}

	// Error propagation.
	input = map[string]string{"k": "$(bad.ns)"}
	_, err = r.ResolveMap(input)
	if err == nil {
		t.Error("ResolveMap() expected error for unknown namespace")
	}
}

func TestMcpxNamespace(t *testing.T) {
	ns := &mcpxNS{projectRoot: "/tmp/test-project"}

	t.Run("project_root", func(t *testing.T) {
		got, err := ns.Resolve("project_root")
		if err != nil {
			t.Fatalf("Resolve(project_root) error = %v", err)
		}
		if got != "/tmp/test-project" {
			t.Errorf("Resolve(project_root) = %q, want %q", got, "/tmp/test-project")
		}
	})

	t.Run("cwd", func(t *testing.T) {
		got, err := ns.Resolve("cwd")
		if err != nil {
			t.Fatalf("Resolve(cwd) error = %v", err)
		}
		cwd, _ := os.Getwd()
		if got != cwd {
			t.Errorf("Resolve(cwd) = %q, want %q", got, cwd)
		}
	})

	t.Run("home", func(t *testing.T) {
		got, err := ns.Resolve("home")
		if err != nil {
			t.Fatalf("Resolve(home) error = %v", err)
		}
		home, _ := os.UserHomeDir()
		if got != home {
			t.Errorf("Resolve(home) = %q, want %q", got, home)
		}
	})

	t.Run("unknown key", func(t *testing.T) {
		_, err := ns.Resolve("nope")
		if err == nil {
			t.Error("Resolve(nope) expected error")
		}
	})
}

func TestEnvNamespace(t *testing.T) {
	ns := &envNS{}

	key := "MCPX_TEST_RESOLVER_VAR"
	os.Setenv(key, "test_value_42")
	defer os.Unsetenv(key)

	got, err := ns.Resolve(key)
	if err != nil {
		t.Fatalf("Resolve(%q) error = %v", key, err)
	}
	if got != "test_value_42" {
		t.Errorf("Resolve(%q) = %q, want %q", key, got, "test_value_42")
	}

	// Unset var returns empty, not error.
	got, err = ns.Resolve("MCPX_NONEXISTENT_VAR_XYZ")
	if err != nil {
		t.Fatalf("Resolve(unset) error = %v", err)
	}
	if got != "" {
		t.Errorf("Resolve(unset) = %q, want empty", got)
	}
}

func TestSysNamespace(t *testing.T) {
	ns := &sysNS{}

	tests := []struct {
		key  string
		want string
	}{
		{"os", runtime.GOOS},
		{"arch", runtime.GOARCH},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, err := ns.Resolve(tt.key)
			if err != nil {
				t.Fatalf("Resolve(%q) error = %v", tt.key, err)
			}
			if got != tt.want {
				t.Errorf("Resolve(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}

	t.Run("unknown key", func(t *testing.T) {
		_, err := ns.Resolve("unknown")
		if err == nil {
			t.Error("Resolve(unknown) expected error")
		}
	})
}

func TestSecretNamespace(t *testing.T) {
	store := &mockSecretStore{data: map[string]string{
		"api_key": "sk-abc123",
	}}
	ns := &secretNS{store: store}

	t.Run("found", func(t *testing.T) {
		got, err := ns.Resolve("api_key")
		if err != nil {
			t.Fatalf("Resolve(api_key) error = %v", err)
		}
		if got != "sk-abc123" {
			t.Errorf("Resolve(api_key) = %q, want %q", got, "sk-abc123")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := ns.Resolve("missing")
		if err == nil {
			t.Error("Resolve(missing) expected error")
		}
	})
}
