package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/codestz/mcpx/internal/config"
	"github.com/codestz/mcpx/internal/mcp"
)

// runGenerate connects to a server, fetches tools (and prompts/resources if supported),
// and generates a concise Markdown reference file for Claude Code.
//
// Uses the cached tools-list path so a configure run populates the schema cache
// for any server we hadn't yet touched — agent's first real call lands warm.
func runGenerate(ctx context.Context, serverName string, sc *config.ServerConfig, global bool, format string) error {
	client, cleanup, err := connectServer(ctx, serverName, sc)
	if err != nil {
		return err
	}
	defer cleanup()

	tools, _, _, err := listToolsCached(ctx, serverName, sc, client)
	if err != nil {
		return fmt.Errorf("list tools: %w", err)
	}

	caps := client.ServerCapabilities()

	var prompts []mcp.Prompt
	if caps.Prompts != nil {
		prompts, _ = client.ListPrompts(ctx)
	}

	var resources []mcp.Resource
	if caps.Resources != nil {
		resources, _ = client.ListResources(ctx)
	}

	content := generateServerMDWithFormat(serverName, tools, prompts, resources, format)

	claudeDir, err := claudeDirectory(global)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("create %s: %w", claudeDir, err)
	}

	fileName := strings.ToUpper(serverName) + ".md"
	filePath := filepath.Join(claudeDir, fileName)

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}
	fmt.Printf("Created %s\n", filePath)

	// Add reference to CLAUDE.md.
	claudeMDPath := filepath.Join(claudeDir, "CLAUDE.md")
	ref := "@" + fileName
	if err := ensureReference(claudeMDPath, ref); err != nil {
		return fmt.Errorf("update %s: %w", claudeMDPath, err)
	}
	fmt.Printf("Updated %s (added %s reference)\n", claudeMDPath, ref)

	// Also reference from MCPX.md if it exists.
	mcpxMDPath := filepath.Join(claudeDir, "MCPX.md")
	if _, err := os.Stat(mcpxMDPath); err == nil {
		if err := ensureReference(mcpxMDPath, ref); err != nil {
			return fmt.Errorf("update %s: %w", mcpxMDPath, err)
		}
	}

	scope := "project"
	if global {
		scope = "global"
	}
	fmt.Printf("\nDone! Claude Code will load %s reference at %s scope.\n", serverName, scope)

	return nil
}

// generateServerMDWithFormat is kept for the configureCmd flag surface but
// now collapses to a single agent-grounded format. The `format` parameter is
// accepted for backward compatibility and ignored.
func generateServerMDWithFormat(serverName string, tools []mcp.Tool, prompts []mcp.Prompt, resources []mcp.Resource, format string) string {
	_ = format
	return generateServerMD(serverName, tools, prompts, resources)
}

// generateServerMD produces an agent-optimized reference for one MCP server.
//
// Design priorities (in order):
//   1. A "tool selector" table at the top — one row per tool, three columns
//      (tool, what it does, required flags). This is the primary surface the
//      agent reads when picking a tool.
//   2. A compact reference at the bottom — every tool's name + required flags
//      on one line, for fast scanning when the selector matches multiple.
//   3. Prompts and resources as terse bullet lists; agents don't reach for
//      them often, so they don't earn full sections.
//
// No per-tool full schema tables. The agent runs `mcpx <server> <tool> --help`
// for the full schema or `--example` for a JSON skeleton when actually calling.
func generateServerMD(serverName string, tools []mcp.Tool, prompts []mcp.Prompt, resources []mcp.Resource) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# %s\n\n", serverName))
	b.WriteString(fmt.Sprintf("`mcpx %s <tool> --flags` — %d tools. Use `--help` / `--example` / `--validate-args` per tool.\n\n",
		serverName, len(tools)))

	b.WriteString("## Tool selector\n\n")
	b.WriteString("| Tool | What it does | Required |\n")
	b.WriteString("|---|---|---|\n")
	for _, t := range tools {
		desc := firstSentence(t.Description)
		if desc == "" {
			desc = "(no description)"
		}
		desc = strings.ReplaceAll(desc, "|", "\\|")
		req := requiredFlagList(t)
		b.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n", t.Name, desc, req))
	}
	b.WriteString("\n")

	if hasEditTools(tools) {
		b.WriteString("## Edit-tool safety\n\n")
		b.WriteString("For `replace_symbol_body`, body must start AFTER the declaration keyword — the keyword is preserved by the server. mcpx warns on duplicates and (for `.go`) re-parses the file post-edit.\n\n")
		b.WriteString("| Language | ✓ correct | ✗ wrong |\n")
		b.WriteString("|---|---|---|\n")
		b.WriteString("| Go | `Foo struct{...}` / `Foo() error {...}` | `type Foo...` / `func Foo...` |\n")
		b.WriteString("| Python | `foo():\\n    ...` | `def foo():` |\n")
		b.WriteString("| TS / JS | `bar() { ... }` | `function bar()` / `class Bar` |\n")
		b.WriteString("| Rust | `foo() { ... }` | `fn foo()` / `struct Foo` |\n\n")
	}

	if hasSymbolSearch(tools) {
		b.WriteString("## Notes\n\n")
		b.WriteString("- Symbol-aware lookups (`find_symbol`) are faster and more accurate than text search (`search_for_pattern`) for code symbols — prefer them.\n\n")
	}

	b.WriteString("## Compact reference\n\n")
	for _, t := range tools {
		propNames := sortedPropNames(t)
		req := map[string]bool{}
		for _, r := range t.InputSchema.Required {
			req[r] = true
		}
		var parts []string
		for _, name := range propNames {
			prop := t.InputSchema.Properties[name]
			entry := fmt.Sprintf("--%s <%s>", name, flagTypeLabel(prop.Type))
			if req[name] {
				entry += "*"
			}
			parts = append(parts, entry)
		}
		if len(parts) == 0 {
			b.WriteString(fmt.Sprintf("- `%s`\n", t.Name))
		} else {
			b.WriteString(fmt.Sprintf("- `%s` %s\n", t.Name, strings.Join(parts, " ")))
		}
	}
	b.WriteString("\n`*` = required\n")

	if len(prompts) > 0 {
		b.WriteString(fmt.Sprintf("\n## Prompts (%d)\n\n", len(prompts)))
		b.WriteString(fmt.Sprintf("`mcpx %s prompt <name> [--arg value ...]`\n\n", serverName))
		for _, p := range prompts {
			line := fmt.Sprintf("- `%s`", p.Name)
			if p.Description != "" {
				line += " — " + firstSentence(p.Description)
			}
			b.WriteString(line + "\n")
		}
	}

	if len(resources) > 0 {
		b.WriteString(fmt.Sprintf("\n## Resources (%d)\n\n", len(resources)))
		b.WriteString(fmt.Sprintf("`mcpx %s resource read <uri>`\n\n", serverName))
		for _, r := range resources {
			line := fmt.Sprintf("- `%s`", r.URI)
			if r.Name != "" {
				line += " — " + r.Name
			}
			b.WriteString(line + "\n")
		}
	}

	return b.String()
}

// requiredFlagList renders a tool's required flags as a compact comma string
// for the selector table. Empty when no flags are required.
func requiredFlagList(t mcp.Tool) string {
	if len(t.InputSchema.Required) == 0 {
		return "—"
	}
	out := make([]string, 0, len(t.InputSchema.Required))
	for _, r := range t.InputSchema.Required {
		out = append(out, "`--"+r+"`")
	}
	return strings.Join(out, " ")
}

// sortedPropNames returns a tool's input property names in stable order.
func sortedPropNames(t mcp.Tool) []string {
	out := make([]string, 0, len(t.InputSchema.Properties))
	for name := range t.InputSchema.Properties {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// hasEditTools detects servers that expose body-mutation tools so we can
// emit the keyword-trap reminder in the per-server doc.
func hasEditTools(tools []mcp.Tool) bool {
	for _, t := range tools {
		if bodyMutatingTools[t.Name] {
			return true
		}
	}
	return false
}

// hasSymbolSearch detects servers that expose both symbol-aware and text-based
// search tools, so the doc can advise the agent to prefer the structured one.
func hasSymbolSearch(tools []mcp.Tool) bool {
	hasSym, hasText := false, false
	for _, t := range tools {
		switch t.Name {
		case "find_symbol":
			hasSym = true
		case "search_for_pattern":
			hasText = true
		}
	}
	return hasSym && hasText
}

// firstSentence returns the first sentence of a string (up to first period+space or newline).
func firstSentence(s string) string {
	// Cut at first newline.
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	// Cut at first ". " (sentence end).
	if i := strings.Index(s, ". "); i >= 0 {
		s = s[:i+1]
	}
	// Limit length.
	if len(s) > 120 {
		s = s[:117] + "..."
	}
	return s
}

// compactDesc shortens a description for table display.
func compactDesc(s string) string {
	// Remove newlines.
	s = strings.ReplaceAll(s, "\n", " ")
	// Collapse spaces.
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	s = strings.TrimSpace(s)
	if len(s) > 80 {
		s = s[:77] + "..."
	}
	return s
}

