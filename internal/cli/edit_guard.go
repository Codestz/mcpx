package cli

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// bodyMutatingTools are MCP tools that overwrite a file region with caller-supplied
// body text. Adding new entries is safe: the guards no-op when arguments don't
// match what they expect.
var bodyMutatingTools = map[string]bool{
	"replace_symbol_body":  true,
	"insert_after_symbol":  true,
	"insert_before_symbol": true,
}

// langKeywords maps file extensions to the set of declaration keywords whose
// presence at the start of a `replace_symbol_body` payload almost certainly
// indicates duplication of the existing symbol's keyword.
//
// Logic for each entry: serena preserves the symbol's `<keyword> <name>` prefix
// and only swaps in the body's content after it; if the body itself starts
// with one of these keywords the file ends up with `<keyword> <keyword> Name`.
var langKeywords = map[string][]string{
	".go":   {"type ", "func "},
	".py":   {"def ", "async def ", "class "},
	".pyi":  {"def ", "async def ", "class "},
	".ts":   {"function ", "class ", "interface ", "type ", "enum "},
	".tsx":  {"function ", "class ", "interface ", "type ", "enum "},
	".js":   {"function ", "class "},
	".jsx":  {"function ", "class "},
	".mjs":  {"function ", "class "},
	".rs":   {"fn ", "struct ", "enum ", "trait ", "impl "},
	".rb":   {"def ", "class ", "module "},
	".java": {"class ", "interface ", "enum ", "void ", "public ", "private ", "protected "},
	".kt":   {"fun ", "class ", "interface ", "object "},
	".swift": {"func ", "class ", "struct ", "enum ", "protocol "},
}

// preCallEditGuard warns when arguments to a body-mutating tool match a known
// failure mode: body starts with the same declaration keyword that the
// symbol-aware MCP server is going to preserve, producing duplicated keywords
// and a syntax error.
//
// Warns to stderr; never blocks the call. Insert variants are skipped because
// they legitimately accept whole new declarations.
func preCallEditGuard(serverName, toolName string, args map[string]any) {
	if toolName != "replace_symbol_body" {
		return
	}
	body, _ := args["body"].(string)
	rel, _ := args["relative_path"].(string)
	if body == "" || rel == "" {
		return
	}
	keywords, ok := langKeywords[strings.ToLower(filepath.Ext(rel))]
	if !ok {
		return
	}
	trimmed := strings.TrimLeft(body, " \t\n")
	for _, kw := range keywords {
		if strings.HasPrefix(trimmed, kw) {
			fmt.Fprintf(os.Stderr,
				"mcpx: warning: body for %s.%s begins with %q. The replacement "+
					"region starts AFTER the symbol's keyword — your %q will be "+
					"duplicated and the file will not parse. Strip the leading keyword.\n",
				serverName, toolName, strings.TrimSpace(kw), strings.TrimSpace(kw))
			return
		}
	}
}

// postCallEditGuard validates the file targeted by a body-mutating tool. For Go
// (.go) we have `go/parser` in stdlib; other languages would require shelling
// out to language-native compilers (python -m py_compile, node --check, etc.)
// which adds external-dependency risk — we keep this in-process for now.
//
// Returns nil for non-edit tools, missing paths, non-Go files, and successful
// parses. Returns a wrapped parse error otherwise so the caller can surface
// it as a tool failure.
func postCallEditGuard(serverName, toolName string, args map[string]any, projectRoot string) error {
	if !bodyMutatingTools[toolName] {
		return nil
	}
	rel, _ := args["relative_path"].(string)
	if rel == "" {
		return nil
	}
	ext := strings.ToLower(filepath.Ext(rel))
	if ext != ".go" {
		// Other languages get the pre-call warning but no post-call check.
		return nil
	}

	full := rel
	if !filepath.IsAbs(rel) {
		root := projectRoot
		if root == "" {
			cwd, _ := os.Getwd()
			root = cwd
		}
		full = filepath.Join(root, rel)
	}

	if _, err := os.Stat(full); err != nil {
		return nil
	}

	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, full, nil, parser.AllErrors); err != nil {
		return fmt.Errorf("post-edit parse check failed for %s\n  %v\n  "+
			"the body sent to %s may have duplicated the symbol's declaration "+
			"keyword (e.g. `type type Foo`); strip the leading `type`/`func`",
			rel, summarizeParseErr(err), toolName)
	}
	return nil
}

// summarizeParseErr trims Go's verbose multi-line parse error to the most
// useful single line for an end-user message.
func summarizeParseErr(err error) string {
	s := err.Error()
	if i := strings.Index(s, "\n"); i > 0 {
		return s[:i]
	}
	return s
}
