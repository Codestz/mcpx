package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/codestz/mcpx/internal/mcp"
)

// generateExample builds a JSON skeleton object showing every flag a tool
// accepts. Required flags get a real placeholder ("<string>" / 0 / false /
// matching enum); optional flags use the schema default when present.
//
// Designed for AI agents that need to construct a valid `--stdin` payload
// quickly without scanning prose descriptions.
func generateExample(tool *mcp.Tool) map[string]any {
	required := map[string]bool{}
	for _, r := range tool.InputSchema.Required {
		required[r] = true
	}

	out := map[string]any{}
	for name, prop := range tool.InputSchema.Properties {
		if required[name] {
			out[name] = placeholderFor(prop)
		} else if prop.Default != nil {
			out[name] = prop.Default
		} else if len(prop.Enum) > 0 {
			out[name] = prop.Enum[0]
		} else {
			out[name] = placeholderFor(prop)
		}
	}
	return out
}

func placeholderFor(p mcp.PropertySchema) any {
	if len(p.Enum) > 0 {
		return p.Enum[0]
	}
	switch p.Type {
	case "string":
		return "<string>"
	case "integer":
		return 0
	case "number":
		return 0.0
	case "boolean":
		return false
	case "array":
		if p.Items != nil {
			return []any{placeholderFor(*p.Items)}
		}
		return []any{}
	case "object":
		return map[string]any{}
	default:
		return nil
	}
}

// printExample prints the example JSON. Pretty-printed by default; --json mode
// strips indentation for piping into other tools.
func printExample(tool *mcp.Tool, jsonMode bool) {
	ex := generateExample(tool)
	enc := json.NewEncoder(stdoutWriter())
	enc.SetEscapeHTML(false)
	if !jsonMode {
		enc.SetIndent("", "  ")
	}
	_ = enc.Encode(ex)
}

func stdoutWriter() *stdoutWriterT { return &stdoutWriterT{} }

type stdoutWriterT struct{}

func (*stdoutWriterT) Write(p []byte) (int, error) {
	return fmt.Print(string(p))
}

// validationIssue describes one problem with a tool-call arguments object.
type validationIssue struct {
	Field    string
	Got      string
	Expected string
	Hint     string
}

func (v validationIssue) String() string {
	parts := []string{fmt.Sprintf("--%s", v.Field)}
	if v.Expected != "" {
		parts = append(parts, "expected "+v.Expected)
	}
	if v.Got != "" {
		parts = append(parts, "got "+v.Got)
	}
	s := strings.Join(parts, ": ")
	if v.Hint != "" {
		s += "  — " + v.Hint
	}
	return s
}

// validateArgs checks args against a tool's normalized schema.
// Returns nil when everything passes, otherwise a list of problems.
func validateArgs(tool *mcp.Tool, args map[string]any) []validationIssue {
	var issues []validationIssue

	// Required check.
	for _, req := range tool.InputSchema.Required {
		if _, ok := args[req]; !ok {
			issues = append(issues, validationIssue{
				Field: req, Expected: "required field", Got: "missing",
			})
		}
	}

	// Type + enum check on supplied values.
	names := make([]string, 0, len(args))
	for k := range args {
		names = append(names, k)
	}
	sort.Strings(names)

	for _, name := range names {
		prop, defined := tool.InputSchema.Properties[name]
		if !defined {
			issues = append(issues, validationIssue{
				Field: name, Expected: "known flag", Got: "unknown",
				Hint: nearestProperty(name, tool.InputSchema.Properties),
			})
			continue
		}
		if !typeMatches(prop, args[name]) {
			issues = append(issues, validationIssue{
				Field: name,
				Expected: prop.Type,
				Got:      describeValue(args[name]),
			})
		}
		if len(prop.Enum) > 0 && !inEnum(args[name], prop.Enum) {
			vals := make([]string, len(prop.Enum))
			for i, v := range prop.Enum {
				vals[i] = fmt.Sprintf("%v", v)
			}
			issues = append(issues, validationIssue{
				Field: name,
				Expected: "one of [" + strings.Join(vals, ", ") + "]",
				Got:      describeValue(args[name]),
			})
		}
	}
	return issues
}

func typeMatches(p mcp.PropertySchema, v any) bool {
	if v == nil {
		return p.Nullable
	}
	switch p.Type {
	case "string":
		_, ok := v.(string)
		return ok
	case "integer":
		switch v.(type) {
		case int, int32, int64, float64:
			return true
		}
		return false
	case "number":
		switch v.(type) {
		case float32, float64, int, int32, int64:
			return true
		}
		return false
	case "boolean":
		_, ok := v.(bool)
		return ok
	case "array":
		_, ok := v.([]any)
		return ok
	case "object":
		_, ok := v.(map[string]any)
		return ok
	case "any", "":
		return true
	}
	return true
}

func inEnum(v any, enum []any) bool {
	for _, e := range enum {
		if fmt.Sprintf("%v", e) == fmt.Sprintf("%v", v) {
			return true
		}
	}
	return false
}

func describeValue(v any) string {
	if v == nil {
		return "null"
	}
	switch v.(type) {
	case string:
		return "string"
	case bool:
		return "boolean"
	case int, int32, int64:
		return "integer"
	case float32, float64:
		return "number"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	}
	return fmt.Sprintf("%T", v)
}
