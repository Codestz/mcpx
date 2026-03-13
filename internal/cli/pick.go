package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/codestz/mcpx/internal/mcp"
)

// pickField extracts a dot-separated field path from a CallResult's text content.
// String values are returned raw; other types are JSON-marshaled.
func pickField(result *mcp.CallResult, path string) (string, error) {
	// Concatenate all text content blocks.
	var text strings.Builder
	for _, c := range result.Content {
		if c.Type == "text" {
			text.WriteString(c.Text)
		}
	}

	raw := text.String()
	var data any
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return "", fmt.Errorf("result is not JSON: %w", err)
	}

	// Walk the dot-separated path.
	parts := strings.Split(path, ".")
	current := data
	for _, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			val, ok := v[part]
			if !ok {
				return "", fmt.Errorf("key %q not found", part)
			}
			current = val
		case []any:
			idx, err := strconv.Atoi(part)
			if err != nil {
				return "", fmt.Errorf("expected integer index for array, got %q", part)
			}
			if idx < 0 || idx >= len(v) {
				return "", fmt.Errorf("index %d out of range (len %d)", idx, len(v))
			}
			current = v[idx]
		default:
			return "", fmt.Errorf("cannot traverse into %T at %q", current, part)
		}
	}

	// String values returned raw, others JSON-marshaled.
	if s, ok := current.(string); ok {
		return s, nil
	}
	out, err := json.Marshal(current)
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}
	return string(out), nil
}
