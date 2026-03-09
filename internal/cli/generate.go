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

// runGenerate connects to a server, fetches tools, and generates a concise
// Markdown reference file for Claude Code.
func runGenerate(ctx context.Context, serverName string, sc *config.ServerConfig, global bool, format string) error {
	client, cleanup, err := connectServer(ctx, serverName, sc)
	if err != nil {
		return err
	}
	defer cleanup()

	tools, err := client.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("list tools: %w", err)
	}

	content := generateServerMDWithFormat(serverName, tools, format)

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

// generateServerMDWithFormat dispatches to the right format generator.
func generateServerMDWithFormat(serverName string, tools []mcp.Tool, format string) string {
	if format == "compact" {
		return generateServerMDCompact(serverName, tools)
	}
	return generateServerMD(serverName, tools)
}

// generateServerMD creates a concise reference for a server's tools.
// Optimized for AI consumption: compact, all flags visible, ~100 lines max.
func generateServerMD(serverName string, tools []mcp.Tool) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# %s — mcpx tool reference\n\n", serverName))
	b.WriteString(fmt.Sprintf("Server with %d tools. Call via: `mcpx %s <tool> --flags`\n\n", len(tools), serverName))

	for _, t := range tools {
		b.WriteString(fmt.Sprintf("## %s\n", t.Name))

		if t.Description != "" {
			// First sentence only for brevity.
			desc := firstSentence(t.Description)
			b.WriteString(fmt.Sprintf("%s\n", desc))
		}

		if len(t.InputSchema.Properties) == 0 {
			b.WriteString("```\nmcpx " + serverName + " " + t.Name + "\n```\n\n")
			continue
		}

		required := make(map[string]bool)
		for _, r := range t.InputSchema.Required {
			required[r] = true
		}

		propNames := make([]string, 0, len(t.InputSchema.Properties))
		for name := range t.InputSchema.Properties {
			propNames = append(propNames, name)
		}
		sort.Strings(propNames)

		// Build example command with required flags.
		var exampleParts []string
		exampleParts = append(exampleParts, "mcpx", serverName, t.Name)

		b.WriteString("| Flag | Type | Req | Description |\n")
		b.WriteString("|---|---|---|---|\n")

		for _, name := range propNames {
			prop := t.InputSchema.Properties[name]
			req := ""
			if required[name] {
				req = "yes"
				exampleParts = append(exampleParts, fmt.Sprintf("--%s <val>", name))
			}
			desc := compactDesc(prop.Description)
			if prop.Default != nil {
				desc += fmt.Sprintf(" (default: %v)", prop.Default)
			}
			b.WriteString(fmt.Sprintf("| `--%s` | %s | %s | %s |\n",
				name, flagTypeLabel(prop.Type), req, desc))
		}

		b.WriteString(fmt.Sprintf("```\n%s\n```\n\n", strings.Join(exampleParts, " ")))
	}

	return b.String()
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

// generateServerMDCompact creates a minimal one-line-per-tool reference.
// Optimized for maximum token efficiency: ~50% smaller than table format.
func generateServerMDCompact(serverName string, tools []mcp.Tool) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# %s (%d tools)\n\n", serverName, len(tools)))
	b.WriteString(fmt.Sprintf("Usage: `mcpx %s <tool> --flags`\n\n", serverName))

	for _, t := range tools {
		required := make(map[string]bool)
		for _, r := range t.InputSchema.Required {
			required[r] = true
		}

		// Tool name + short description
		desc := ""
		if t.Description != "" {
			desc = " — " + firstSentence(t.Description)
		}
		b.WriteString(fmt.Sprintf("**%s**%s\n", t.Name, desc))

		if len(t.InputSchema.Properties) > 0 {
			propNames := make([]string, 0, len(t.InputSchema.Properties))
			for name := range t.InputSchema.Properties {
				propNames = append(propNames, name)
			}
			sort.Strings(propNames)

			var flags []string
			for _, name := range propNames {
				prop := t.InputSchema.Properties[name]
				entry := fmt.Sprintf("--%s <%s>", name, flagTypeLabel(prop.Type))
				if required[name] {
					entry += " *"
				}
				flags = append(flags, entry)
			}
			b.WriteString("  " + strings.Join(flags, ", ") + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("`*` = required\n")
	return b.String()
}
