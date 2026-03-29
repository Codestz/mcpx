package cli

import (
	"testing"

	"github.com/codestz/mcpx/internal/mcp"
)

func TestPickField(t *testing.T) {
	tests := []struct {
		name    string
		result  *mcp.CallResult
		path    string
		want    string
		wantErr bool
	}{
		{
			name: "simple object pick",
			result: &mcp.CallResult{
				Content: []mcp.Content{{Type: "text", Text: `{"name":"Alice","age":30}`}},
			},
			path: "name",
			want: "Alice",
		},
		{
			name: "nested pick",
			result: &mcp.CallResult{
				Content: []mcp.Content{{Type: "text", Text: `{"user":{"name":"Bob"}}`}},
			},
			path: "user.name",
			want: "Bob",
		},
		{
			name: "array index pick",
			result: &mcp.CallResult{
				Content: []mcp.Content{{Type: "text", Text: `{"items":["a","b","c"]}`}},
			},
			path: "items.1",
			want: "b",
		},
		{
			name: "non-string value marshaled",
			result: &mcp.CallResult{
				Content: []mcp.Content{{Type: "text", Text: `{"count":42}`}},
			},
			path: "count",
			want: "42",
		},
		{
			name: "nested object marshaled",
			result: &mcp.CallResult{
				Content: []mcp.Content{{Type: "text", Text: `{"data":{"x":1}}`}},
			},
			path: "data",
			want: `{"x":1}`,
		},
		{
			name: "non-JSON error",
			result: &mcp.CallResult{
				Content: []mcp.Content{{Type: "text", Text: `not json`}},
			},
			path:    "foo",
			wantErr: true,
		},
		{
			name: "missing key error",
			result: &mcp.CallResult{
				Content: []mcp.Content{{Type: "text", Text: `{"name":"Alice"}`}},
			},
			path:    "missing",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pickField(tt.result, tt.path)
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
