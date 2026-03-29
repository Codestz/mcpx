package security

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codestz/mcpx/v2/internal/config"
)

func TestEvaluate_NoSecurity(t *testing.T) {
	e := NewEvaluator("test", nil, nil)
	r := e.Evaluate("any_tool", map[string]any{"key": "value"})
	if r.Action != ActionAllow {
		t.Errorf("expected allow, got %v", r.Action)
	}
}

func TestEvaluate_ReadOnlyMode(t *testing.T) {
	tests := []struct {
		name   string
		tool   string
		want   Action
	}{
		{"read tool allowed", "find_symbol", ActionAllow},
		{"list tool allowed", "list_dir", ActionAllow},
		{"search allowed", "search_for_pattern", ActionAllow},
		{"replace denied", "replace_symbol_body", ActionDeny},
		{"insert denied", "insert_after_symbol", ActionDeny},
		{"delete denied", "delete_memory", ActionDeny},
		{"rename denied", "rename_symbol", ActionDeny},
		{"create denied", "create_issue", ActionDeny},
		{"write denied", "write_memory", ActionDeny},
		{"execute denied", "execute", ActionDeny},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEvaluator("test", nil, &config.ServerSecurity{Mode: "read-only"})
			r := e.Evaluate(tt.tool, nil)
			if r.Action != tt.want {
				t.Errorf("tool %q: got %v, want %v", tt.tool, r.Action, tt.want)
			}
		})
	}
}

func TestEvaluate_AllowedTools(t *testing.T) {
	e := NewEvaluator("test", nil, &config.ServerSecurity{
		AllowedTools: []string{"find_*", "list_*", "search_*", "get_*"},
	})

	tests := []struct {
		tool string
		want Action
	}{
		{"find_symbol", ActionAllow},
		{"list_dir", ActionAllow},
		{"search_for_pattern", ActionAllow},
		{"get_symbols_overview", ActionAllow},
		{"replace_symbol_body", ActionDeny},
		{"delete_memory", ActionDeny},
	}
	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			r := e.Evaluate(tt.tool, nil)
			if r.Action != tt.want {
				t.Errorf("got %v, want %v", r.Action, tt.want)
			}
		})
	}
}

func TestEvaluate_BlockedTools(t *testing.T) {
	e := NewEvaluator("test", nil, &config.ServerSecurity{
		BlockedTools: []string{"delete_*", "drop_*"},
	})

	tests := []struct {
		tool string
		want Action
	}{
		{"find_symbol", ActionAllow},
		{"delete_memory", ActionDeny},
		{"drop_database", ActionDeny},
	}
	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			r := e.Evaluate(tt.tool, nil)
			if r.Action != tt.want {
				t.Errorf("got %v, want %v", r.Action, tt.want)
			}
		})
	}
}

func TestEvaluate_ArgDenyPattern(t *testing.T) {
	e := NewEvaluator("test", nil, &config.ServerSecurity{
		Policies: []config.Policy{
			{
				Name:    "no-traversal",
				Match:   config.PolicyMatch{Args: map[string]config.ArgRule{"relative_path": {DenyPattern: `\.\.\/`}}},
				Action:  "deny",
				Message: "Path traversal blocked",
			},
		},
	})

	tests := []struct {
		name string
		args map[string]any
		want Action
	}{
		{"clean path", map[string]any{"relative_path": "src/main.go"}, ActionAllow},
		{"traversal", map[string]any{"relative_path": "../../../etc/passwd"}, ActionDeny},
		{"no matching arg", map[string]any{"other": "value"}, ActionAllow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := e.Evaluate("any_tool", tt.args)
			if r.Action != tt.want {
				t.Errorf("got %v, want %v", r.Action, tt.want)
			}
		})
	}
}

func TestEvaluate_ArgAllowPrefix(t *testing.T) {
	e := NewEvaluator("test", nil, &config.ServerSecurity{
		Policies: []config.Policy{
			{
				Name: "restrict-paths",
				Match: config.PolicyMatch{
					Tools: []string{"*"},
					Args:  map[string]config.ArgRule{"relative_path": {AllowPrefix: []string{"src/", "internal/", "cmd/", "./"}}},
				},
				Action:  "deny",
				Message: "Access restricted",
			},
		},
	})

	tests := []struct {
		path string
		want Action
	}{
		{"src/main.go", ActionAllow},
		{"internal/config/config.go", ActionAllow},
		{"cmd/mcpx/main.go", ActionAllow},
		{"./file.go", ActionAllow},
		{"vendor/lib.go", ActionDeny},
		{"/etc/passwd", ActionDeny},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			r := e.Evaluate("find_symbol", map[string]any{"relative_path": tt.path})
			if r.Action != tt.want {
				t.Errorf("path %q: got %v, want %v", tt.path, r.Action, tt.want)
			}
		})
	}
}

func TestEvaluate_ContentDenyPattern(t *testing.T) {
	e := NewEvaluator("postgres", nil, &config.ServerSecurity{
		Policies: []config.Policy{
			{
				Name: "no-mutations",
				Match: config.PolicyMatch{
					Tools: []string{"query"},
					Content: &config.ContentMatch{
						Target:      "args.sql",
						DenyPattern: `(?i)\b(INSERT|UPDATE|DELETE|DROP|TRUNCATE|ALTER)\b`,
					},
				},
				Action:  "deny",
				Message: "Mutation queries blocked",
			},
		},
	})

	tests := []struct {
		name string
		sql  string
		want Action
	}{
		{"select", "SELECT * FROM users LIMIT 10", ActionAllow},
		{"insert", "INSERT INTO users (name) VALUES ('test')", ActionDeny},
		{"delete", "DELETE FROM users WHERE id = 1", ActionDeny},
		{"drop", "DROP TABLE users", ActionDeny},
		{"mixed case", "drop table users", ActionDeny},
		{"truncate", "TRUNCATE users", ActionDeny},
		{"alter", "ALTER TABLE users ADD COLUMN age INT", ActionDeny},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := e.Evaluate("query", map[string]any{"sql": tt.sql})
			if r.Action != tt.want {
				t.Errorf("sql %q: got %v, want %v", tt.sql, r.Action, tt.want)
			}
		})
	}

	// Tool name mismatch — policy should not apply.
	r := e.Evaluate("list_tables", map[string]any{"sql": "DROP TABLE x"})
	if r.Action != ActionAllow {
		t.Errorf("non-matching tool: got %v, want allow", r.Action)
	}
}

func TestEvaluate_ContentRequirePattern(t *testing.T) {
	e := NewEvaluator("postgres", nil, &config.ServerSecurity{
		Policies: []config.Policy{
			{
				Name: "require-limit",
				Match: config.PolicyMatch{
					Tools: []string{"query"},
					Content: &config.ContentMatch{
						Target:         "args.sql",
						RequirePattern: `(?i)\bLIMIT\b`,
						When:           `(?i)^\s*SELECT`,
					},
				},
				Action:  "warn",
				Message: "SELECT without LIMIT",
			},
		},
	})

	tests := []struct {
		name string
		sql  string
		want Action
	}{
		{"select with limit", "SELECT * FROM users LIMIT 10", ActionAllow},
		{"select without limit", "SELECT * FROM users", ActionWarn},
		{"insert (when doesn't match)", "INSERT INTO users VALUES (1)", ActionAllow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := e.Evaluate("query", map[string]any{"sql": tt.sql})
			if r.Action != tt.want {
				t.Errorf("got %v, want %v", r.Action, tt.want)
			}
		})
	}
}

func TestEvaluate_GlobalAndServerPolicies(t *testing.T) {
	global := &config.SecurityConfig{
		Enabled: true,
		Global: config.GlobalSecurity{
			Policies: []config.Policy{
				{
					Name:    "global-no-traversal",
					Match:   config.PolicyMatch{Args: map[string]config.ArgRule{"relative_path": {DenyPattern: `\.\.\/`}}},
					Action:  "deny",
					Message: "Global: path traversal blocked",
				},
			},
		},
	}
	server := &config.ServerSecurity{
		Policies: []config.Policy{
			{
				Name:    "server-restrict",
				Match:   config.PolicyMatch{Args: map[string]config.ArgRule{"relative_path": {AllowPrefix: []string{"src/"}}}},
				Action:  "deny",
				Message: "Server: restricted to src/",
			},
		},
	}

	e := NewEvaluator("test", global, server)

	// Global policy triggers first (traversal).
	r := e.Evaluate("find_symbol", map[string]any{"relative_path": "../etc/passwd"})
	if r.Action != ActionDeny {
		t.Errorf("traversal: got %v, want deny", r.Action)
	}
	if r.PolicyName != "global-no-traversal" {
		t.Errorf("expected global policy, got %q", r.PolicyName)
	}

	// Server policy triggers (not in src/).
	r = e.Evaluate("find_symbol", map[string]any{"relative_path": "vendor/lib.go"})
	if r.Action != ActionDeny {
		t.Errorf("vendor: got %v, want deny", r.Action)
	}
	if r.PolicyName != "server-restrict" {
		t.Errorf("expected server policy, got %q", r.PolicyName)
	}

	// Both pass.
	r = e.Evaluate("find_symbol", map[string]any{"relative_path": "src/main.go"})
	if r.Action != ActionAllow {
		t.Errorf("src: got %v, want allow", r.Action)
	}
}

func TestEvaluate_WarnAction(t *testing.T) {
	e := NewEvaluator("test", nil, &config.ServerSecurity{
		Policies: []config.Policy{
			{
				Name:    "warn-large",
				Match:   config.PolicyMatch{Tools: []string{"query"}},
				Action:  "warn",
				Message: "Be careful with queries",
			},
		},
	})

	r := e.Evaluate("query", nil)
	if r.Action != ActionWarn {
		t.Errorf("got %v, want warn", r.Action)
	}
}

func TestEvaluate_ArgGlobPattern(t *testing.T) {
	e := NewEvaluator("test", nil, &config.ServerSecurity{
		Policies: []config.Policy{
			{
				Name:    "no-traversal-any-path",
				Match:   config.PolicyMatch{Args: map[string]config.ArgRule{"*path*": {DenyPattern: `\.\.\/`}}},
				Action:  "deny",
				Message: "Traversal in any path arg",
			},
		},
	})

	// Matches "relative_path" via glob "*path*".
	r := e.Evaluate("tool", map[string]any{"relative_path": "../secret"})
	if r.Action != ActionDeny {
		t.Errorf("got %v, want deny", r.Action)
	}

	// Matches "file_path" via glob.
	r = e.Evaluate("tool", map[string]any{"file_path": "../secret"})
	if r.Action != ActionDeny {
		t.Errorf("got %v, want deny", r.Action)
	}

	// Non-path arg — no match.
	r = e.Evaluate("tool", map[string]any{"name": "../secret"})
	if r.Action != ActionAllow {
		t.Errorf("got %v, want allow", r.Action)
	}
}

// --- Audit logger tests ---

func TestAuditLogger_Log(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")

	logger := NewAuditLogger(path, nil)

	err := logger.Log(AuditEntry{
		Server: "serena",
		Tool:   "find_symbol",
		Args:   map[string]any{"name_path_pattern": "Config"},
		Action: "allowed",
	})
	if err != nil {
		t.Fatalf("Log() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var entry AuditEntry
	if err := json.Unmarshal(data[:len(data)-1], &entry); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if entry.Server != "serena" {
		t.Errorf("server = %q, want serena", entry.Server)
	}
	if entry.Tool != "find_symbol" {
		t.Errorf("tool = %q, want find_symbol", entry.Tool)
	}
	if entry.Timestamp == "" {
		t.Error("timestamp is empty")
	}
}

func TestAuditLogger_Redact(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")

	logger := NewAuditLogger(path, []string{"$(secret.*)"})

	err := logger.Log(AuditEntry{
		Server: "test",
		Tool:   "connect",
		Args: map[string]any{
			"host":     "localhost",
			"password": "hunter2",
			"token":    "sk-abc123",
			"name":     "mydb",
		},
		Action: "allowed",
	})
	if err != nil {
		t.Fatalf("Log() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var entry AuditEntry
	if err := json.Unmarshal(data[:len(data)-1], &entry); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if entry.Args["password"] != "[REDACTED]" {
		t.Errorf("password not redacted: %v", entry.Args["password"])
	}
	if entry.Args["token"] != "[REDACTED]" {
		t.Errorf("token not redacted: %v", entry.Args["token"])
	}
	if entry.Args["host"] != "localhost" {
		t.Errorf("host was redacted: %v", entry.Args["host"])
	}
	if entry.Args["name"] != "mydb" {
		t.Errorf("name was redacted: %v", entry.Args["name"])
	}
}

func TestAuditLogger_MultipleEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")

	logger := NewAuditLogger(path, nil)

	for i := 0; i < 3; i++ {
		if err := logger.Log(AuditEntry{Server: "test", Tool: "tool", Action: "allowed"}); err != nil {
			t.Fatal(err)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}

func TestActionString(t *testing.T) {
	tests := []struct {
		action Action
		want   string
	}{
		{ActionAllow, "allowed"},
		{ActionDeny, "denied"},
		{ActionWarn, "warned"},
		{Action(99), "unknown"},
	}
	for _, tt := range tests {
		if got := ActionString(tt.action); got != tt.want {
			t.Errorf("ActionString(%v) = %q, want %q", tt.action, got, tt.want)
		}
	}
}
