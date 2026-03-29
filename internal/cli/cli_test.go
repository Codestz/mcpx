package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/codestz/mcpx/v2/internal/mcp"
)

func TestParseToolFlags(t *testing.T) {
	tests := []struct {
		name    string
		tool    mcp.Tool
		args    []string
		want    map[string]any
		wantErr bool
	}{
		{
			name: "string flag",
			tool: mcp.Tool{
				Name: "test",
				InputSchema: mcp.InputSchema{
					Type: "object",
					Properties: map[string]mcp.PropertySchema{
						"name": {Type: "string", Description: "The name"},
					},
				},
			},
			args: []string{"--name", "hello"},
			want: map[string]any{"name": "hello"},
		},
		{
			name: "integer flag",
			tool: mcp.Tool{
				Name: "test",
				InputSchema: mcp.InputSchema{
					Type: "object",
					Properties: map[string]mcp.PropertySchema{
						"count": {Type: "integer", Description: "Count"},
					},
				},
			},
			args: []string{"--count", "42"},
			want: map[string]any{"count": int64(42)},
		},
		{
			name: "boolean flag",
			tool: mcp.Tool{
				Name: "test",
				InputSchema: mcp.InputSchema{
					Type: "object",
					Properties: map[string]mcp.PropertySchema{
						"verbose": {Type: "boolean", Description: "Verbose"},
					},
				},
			},
			args: []string{"--verbose"},
			want: map[string]any{"verbose": true},
		},
		{
			name: "number flag",
			tool: mcp.Tool{
				Name: "test",
				InputSchema: mcp.InputSchema{
					Type: "object",
					Properties: map[string]mcp.PropertySchema{
						"ratio": {Type: "number", Description: "Ratio"},
					},
				},
			},
			args: []string{"--ratio", "3.14"},
			want: map[string]any{"ratio": 3.14},
		},
		{
			name: "array flag with JSON",
			tool: mcp.Tool{
				Name: "test",
				InputSchema: mcp.InputSchema{
					Type: "object",
					Properties: map[string]mcp.PropertySchema{
						"tags": {Type: "array", Description: "Tags"},
					},
				},
			},
			args: []string{"--tags", `["a","b","c"]`},
			want: map[string]any{"tags": []any{"a", "b", "c"}},
		},
		{
			name: "array flag with comma-separated",
			tool: mcp.Tool{
				Name: "test",
				InputSchema: mcp.InputSchema{
					Type: "object",
					Properties: map[string]mcp.PropertySchema{
						"tags": {Type: "array", Description: "Tags"},
					},
				},
			},
			args: []string{"--tags", "a,b,c"},
			want: map[string]any{"tags": []any{"a", "b", "c"}},
		},
		{
			name: "unset flags omitted",
			tool: mcp.Tool{
				Name: "test",
				InputSchema: mcp.InputSchema{
					Type: "object",
					Properties: map[string]mcp.PropertySchema{
						"name":  {Type: "string", Description: "Name"},
						"count": {Type: "integer", Description: "Count"},
					},
				},
			},
			args: []string{"--name", "hello"},
			want: map[string]any{"name": "hello"},
		},
		{
			name: "required flag missing",
			tool: mcp.Tool{
				Name: "test",
				InputSchema: mcp.InputSchema{
					Type: "object",
					Properties: map[string]mcp.PropertySchema{
						"name": {Type: "string", Description: "Name"},
					},
					Required: []string{"name"},
				},
			},
			args:    []string{},
			wantErr: true,
		},
		{
			name: "object flag",
			tool: mcp.Tool{
				Name: "test",
				InputSchema: mcp.InputSchema{
					Type: "object",
					Properties: map[string]mcp.PropertySchema{
						"config": {Type: "object", Description: "Config"},
					},
				},
			},
			args: []string{"--config", `{"key":"value"}`},
			want: map[string]any{"config": map[string]any{"key": "value"}},
		},
		{
			name: "string flag with @file",
			tool: mcp.Tool{
				Name: "test",
				InputSchema: mcp.InputSchema{
					Type: "object",
					Properties: map[string]mcp.PropertySchema{
						"body": {Type: "string", Description: "Body"},
					},
				},
			},
			args: []string{"--body", "@TESTFILE"}, // placeholder, overridden in test
			want: map[string]any{"body": "file content here"},
		},
	}

	// Set up temp file for the @file test.
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "testbody.txt")
	if err := os.WriteFile(tmpFile, []byte("file content here\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Patch the @file test case with the real path.
	for i := range tests {
		if tests[i].name == "string flag with @file" {
			tests[i].args = []string{"--body", "@" + tmpFile}
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseToolFlags(&tt.tool, tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d args, want %d: %v", len(got), len(tt.want), got)
			}
			for k, wantV := range tt.want {
				gotV, ok := got[k]
				if !ok {
					t.Errorf("missing key %q", k)
					continue
				}
				if !deepEqual(gotV, wantV) {
					t.Errorf("key %q: got %v (%T), want %v (%T)", k, gotV, gotV, wantV, wantV)
				}
			}
		})
	}
}

func TestResolveStringValue(t *testing.T) {
	// Create a temp file for @file tests.
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "data.txt")
	if err := os.WriteFile(tmpFile, []byte("hello from file\n"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		val     string
		want    string
		wantErr bool
	}{
		{"literal string", "hello", "hello", false},
		{"@file reads file", "@" + tmpFile, "hello from file", false},
		{"@nonexistent errors", "@/tmp/nonexistent_mcpx_test_file", "", true},
		{"bare @ is literal", "@", "@", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveStringValue(tt.val, "test")
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseToolFlagsPartial(t *testing.T) {
	tool := &mcp.Tool{
		Name: "test",
		InputSchema: mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.PropertySchema{
				"name": {Type: "string", Description: "Name"},
				"body": {Type: "string", Description: "Body"},
			},
			Required: []string{"name", "body"},
		},
	}

	// parseToolFlagsPartial should not error on missing required flags.
	got, err := parseToolFlagsPartial(tool, []string{"--name", "Foo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["name"] != "Foo" {
		t.Errorf("name: got %q, want %q", got["name"], "Foo")
	}
	if _, ok := got["body"]; ok {
		t.Errorf("body should not be set")
	}
}

func TestGlobalOpts_OutputMode(t *testing.T) {
	tests := []struct {
		name string
		opts globalOpts
		want outputMode
	}{
		{"default", globalOpts{}, outputPretty},
		{"json", globalOpts{jsonOutput: true}, outputJSON},
		{"quiet", globalOpts{quiet: true}, outputQuiet},
		{"quiet wins over json", globalOpts{jsonOutput: true, quiet: true}, outputQuiet},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.opts.outputMode(); got != tt.want {
				t.Errorf("outputMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input string
		max   int
		want  string
	}{
		{"short", 10, "short"},
		{"this is a long string", 10, "this is..."},
		{"line1\nline2", 20, "line1"},
		{"", 5, ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := truncate(tt.input, tt.max); got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.want)
			}
		})
	}
}

func TestJoinOr(t *testing.T) {
	tests := []struct {
		input []string
		want  string
	}{
		{nil, ""},
		{[]string{"a"}, "a"},
		{[]string{"a", "b"}, "a or b"},
		{[]string{"a", "b", "c"}, "a, b or c"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := joinOr(tt.input); got != tt.want {
				t.Errorf("joinOr(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// deepEqual handles comparison of any values including slices.
func deepEqual(a, b any) bool {
	switch av := a.(type) {
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !deepEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, v := range av {
			if !deepEqual(v, bv[k]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}
