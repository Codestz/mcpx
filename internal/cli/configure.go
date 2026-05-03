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
	var format string // accepted for backward compatibility; ignored.

	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Set up mcpx for Claude Code (writes MCPX.md + per-server docs)",
		Long: `Generate the agent-facing reference docs that teach Claude Code (and any
mcpx-aware coding agent) how to use the configured MCP servers.

Writes:
  - MCPX.md       composition + discover + edit-safety + exit codes
  - <SERVER>.md   tool-selector table + compact reference, per server
  - CLAUDE.md     adds @MCPX.md and @<SERVER>.md references

Idempotent — re-run after adding/removing servers.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = format // legacy flag

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
			fmt.Printf("\nDone! Claude Code will load mcpx references at %s scope.\n", scope)

			return nil
		},
	}

	cmd.Flags().BoolVar(&global, "global", false, "Configure globally (~/.claude/) instead of project (.claude/)")
	cmd.Flags().StringVar(&format, "format", "default", "Deprecated; format is now agent-optimized by default")
	_ = cmd.Flags().MarkHidden("format")

	return cmd
}

// generateMCPXMD creates the content for MCPX.md — the always-loaded reference
// an AI agent uses to call mcpx-wrapped MCP servers.
//
// Designed for a coding agent reading the file at session start. Tabular,
// composition-first, no human-ops content (caching internals, observability,
// dashboard, doctor — all excluded; they're available via mcpx commands when
// the human needs them, but they don't help the agent pick a tool).
func generateMCPXMD(cfg *config.Config) string {
	var b strings.Builder

	b.WriteString("# mcpx\n\n")
	b.WriteString("Call MCP tools through `mcpx <server> <tool> --flags`. ")
	b.WriteString("Don't load native MCP — use the CLI commands below.\n\n")

	b.WriteString("## Servers\n\n")
	if len(cfg.Servers) == 0 {
		b.WriteString("(none — run `mcpx init`)\n\n")
	} else {
		for name, sc := range cfg.Servers {
			b.WriteString(fmt.Sprintf("- **%s**", name))
			if sc.Daemon {
				b.WriteString(" *(daemon)*")
			}
			if sc.Transport != "" && sc.Transport != "stdio" {
				b.WriteString(fmt.Sprintf(" *(%s)*", sc.Transport))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("## Compose\n\n")
	b.WriteString("| Need | How |\n")
	b.WriteString("|---|---|\n")
	b.WriteString("| Standard call | `mcpx <server> <tool> --flag value` |\n")
	b.WriteString("| Large arg from file | `--body @/path/to/file` |\n")
	b.WriteString("| Read body from stdin | `--body @-` or `--body -` |\n")
	b.WriteString("| Pass full args as JSON | `printf '{...}' \\| mcpx <server> <tool> --stdin` |\n")
	b.WriteString("| Mix stdin + flags | `--stdin --flag value` (flags win) |\n")
	b.WriteString("| Extract one JSON field | `--pick path.to.field` |\n")
	b.WriteString("| Raw JSON output | `--json` |\n")
	b.WriteString("| Per-call timeout | `--timeout 60s` (Go duration) |\n")
	b.WriteString("| Show resolved command | `--dry-run` |\n")
	b.WriteString("| Args skeleton | `mcpx <server> <tool> --example` |\n")
	b.WriteString("| Type-check args | `mcpx <server> <tool> --validate-args ...` |\n\n")

	b.WriteString("## Discover\n\n")
	b.WriteString("| Need | How |\n")
	b.WriteString("|---|---|\n")
	b.WriteString("| Find the right tool by intent | `mcpx find \"<query>\"` |\n")
	b.WriteString("| One-line list of a server's tools | `mcpx <server> --help` |\n")
	b.WriteString("| Full schema for one tool | `mcpx <server> <tool> --help` |\n")
	b.WriteString("| Run many tool calls in parallel | `mcpx batch < calls.jsonl` |\n\n")

	b.WriteString("## Exit codes\n\n")
	b.WriteString("`0` ok · `1` tool error · `2` config · `3` connection · `4` timeout · `5` policy denied · `6` tool not found.\n")

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
