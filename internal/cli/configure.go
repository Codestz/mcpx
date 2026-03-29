package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/codestz/mcpx/internal/config"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func configureCmd() *cobra.Command {
	var global bool
	var format string

	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Set up mcpx for Claude Code (creates MCPX.md + per-server docs)",
		Long: `Configure mcpx for Claude Code integration.

Creates MCPX.md (quick reference), connects to each configured server,
generates per-server tool docs (SERVER.md), and updates CLAUDE.md references.

Formats:
  default  — tables with flag/type/required/description columns
  compact  — one line per tool, flags inline (smaller, ~50% fewer tokens)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if format != "default" && format != "compact" {
				return fmt.Errorf("invalid format %q: must be 'default' or 'compact'", format)
			}

			cfg, _, err := config.Load()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}

			claudeDir, err := claudeDirectory(global)
			if err != nil {
				return err
			}

			// Ensure .claude/ directory exists.
			if err := os.MkdirAll(claudeDir, 0755); err != nil {
				return fmt.Errorf("create %s: %w", claudeDir, err)
			}

			// Generate MCPX.md content.
			mcpxMD := generateMCPXMD(cfg)
			mcpxPath := filepath.Join(claudeDir, "MCPX.md")
			if err := os.WriteFile(mcpxPath, []byte(mcpxMD), 0644); err != nil {
				return fmt.Errorf("write %s: %w", mcpxPath, err)
			}
			fmt.Printf("Created %s\n", mcpxPath)

			// Update CLAUDE.md to reference MCPX.md.
			claudeMDPath := filepath.Join(claudeDir, "CLAUDE.md")
			if err := ensureReference(claudeMDPath, "@MCPX.md"); err != nil {
				return fmt.Errorf("update %s: %w", claudeMDPath, err)
			}
			fmt.Printf("Updated %s (added @MCPX.md reference)\n", claudeMDPath)

			// Generate per-server docs.
			if len(cfg.Servers) > 0 {
				fmt.Println()
				ctx := cmd.Context()
				for name, sc := range cfg.Servers {
					fmt.Printf("Generating docs for %s...\n", color.CyanString(name))
					if err := runGenerate(ctx, name, sc, global, format); err != nil {
						fmt.Printf("  %s: %v (skipped)\n", color.YellowString("warning"), err)
						continue
					}
				}
			}

			scope := "project"
			if global {
				scope = "global"
			}
			fmt.Printf("\nDone! Claude Code will load mcpx references at %s scope (format: %s).\n", scope, format)

			return nil
		},
	}

	cmd.Flags().BoolVar(&global, "global", false, "Configure globally (~/.claude/) instead of project (.claude/)")
	cmd.Flags().StringVar(&format, "format", "default", "Doc format: 'default' (tables) or 'compact' (one-line per tool)")

	return cmd
}

// generateMCPXMD creates the content for MCPX.md.
func generateMCPXMD(cfg *config.Config) string {
	var b strings.Builder

	b.WriteString("# mcpx — MCP Server CLI Proxy\n\n")
	b.WriteString("mcpx wraps MCP servers into CLI tools. Call them via Bash instead of loading schemas into context.\n\n")

	b.WriteString("## Quick Reference\n\n")
	b.WriteString("```bash\n")
	b.WriteString("mcpx list                        # List configured servers\n")
	b.WriteString("mcpx list <server> -v            # List all tools with flags\n")
	b.WriteString("mcpx <server> --help             # Show server tools\n")
	b.WriteString("mcpx <server> <tool> --help      # Show tool flags\n")
	b.WriteString("mcpx <server> <tool> --flags     # Call a tool\n")
	b.WriteString("mcpx <server> <tool> --stdin      # Read args from stdin JSON\n")
	b.WriteString("mcpx <server> <tool> --json       # Output raw JSON\n")
	b.WriteString("mcpx daemon status               # Show running daemons\n")
	b.WriteString("mcpx <server> info               # Show server capabilities\n")
	b.WriteString("mcpx <server> prompt list         # List available prompts\n")
	b.WriteString("mcpx <server> prompt <name> --args # Get a prompt\n")
	b.WriteString("mcpx <server> resource list       # List available resources\n")
	b.WriteString("mcpx <server> resource read <uri> # Read a resource\n")
	b.WriteString("```\n\n")

	b.WriteString("## Configured Servers\n\n")
	if len(cfg.Servers) == 0 {
		b.WriteString("No servers configured. Run `mcpx init` to import from `.mcp.json`.\n")
	} else {
		for name, sc := range cfg.Servers {
			b.WriteString(fmt.Sprintf("- **%s** — `%s`", name, sc.Command))
			if sc.Daemon {
				b.WriteString(" (daemon)")
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n## Usage Pattern\n\n")
	b.WriteString("1. Discover: `mcpx <server> --help` to see available tools\n")
	b.WriteString("2. Inspect: `mcpx <server> <tool> --help` to see flags\n")
	b.WriteString("3. Call: `mcpx <server> <tool> --flag value`\n")
	b.WriteString("4. For long args: `printf '{\"key\":\"value\"}' | mcpx <server> <tool> --stdin`\n")

	b.WriteString("\n## Large Content: @file syntax\n\n")
	b.WriteString("Any string flag accepts `@/path` to read from a file or `@-`/`-` to read from stdin:\n")
	b.WriteString("```bash\n")
	b.WriteString("mcpx <server> <tool> --body @/tmp/code.go   # Read file into --body\n")
	b.WriteString("mcpx <server> <tool> --body @-              # Read stdin into --body\n")
	b.WriteString("mcpx <server> <tool> --body -               # Same (backward compat)\n")
	b.WriteString("```\n")

	b.WriteString("\n## Output Extraction: --pick\n\n")
	b.WriteString("Extract a JSON field from the result without jq:\n")
	b.WriteString("```bash\n")
	b.WriteString("mcpx <server> <tool> --pick field.path      # Dot-separated path\n")
	b.WriteString("mcpx <server> <tool> --pick items.0.name    # Array index access\n")
	b.WriteString("```\n")

	b.WriteString("\n## Timeout Override: --timeout\n\n")
	b.WriteString("Override the default call timeout for a single invocation:\n")
	b.WriteString("```bash\n")
	b.WriteString("mcpx <server> <tool> --timeout 60s          # Go duration format\n")
	b.WriteString("```\n")

	b.WriteString("\n## Stdin Merge\n\n")
	b.WriteString("`--stdin` can be combined with CLI flags. Flags win on conflict:\n")
	b.WriteString("```bash\n")
	b.WriteString("echo '{\"body\":\"content\"}' | mcpx <server> <tool> --stdin --name_path Foo\n")
	b.WriteString("```\n")

	b.WriteString("\n## Tips for AI Agents\n\n")
	b.WriteString("- Use `--body @/tmp/file` for large content to avoid shell escaping\n")
	b.WriteString("- Use `--pick field` instead of piping through jq for single fields\n")
	b.WriteString("- Combine `--stdin` with flags for mixed large+small arguments\n")
	b.WriteString("- Use `--timeout 120s` for long-running operations\n")

	return b.String()
}

// claudeDirectory returns the path to the .claude/ directory.
func claudeDirectory(global bool) (string, error) {
	if global {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("home dir: %w", err)
		}
		return filepath.Join(home, ".claude"), nil
	}

	// Project: find project root, use .claude/ there.
	root, err := findProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".claude"), nil
}

// ensureReference adds a reference line to CLAUDE.md if not already present.
func ensureReference(claudeMDPath, ref string) error {
	var content string

	data, err := os.ReadFile(claudeMDPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		// File doesn't exist — create with reference.
		content = ref + "\n"
	} else {
		content = string(data)
		// Check if reference already exists.
		if strings.Contains(content, ref) {
			return nil // already referenced
		}
		// Append reference.
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += ref + "\n"
	}

	return os.WriteFile(claudeMDPath, []byte(content), 0644)
}
