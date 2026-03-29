package lifecycle

import (
	"context"
	"fmt"
	"strings"

	"github.com/codestz/mcpx/v2/internal/config"
	"github.com/codestz/mcpx/v2/internal/mcp"
)

// RunOnConnect executes lifecycle hooks after a successful MCP connection.
// Hooks run sequentially; if any hook fails, remaining hooks are skipped.
func RunOnConnect(ctx context.Context, client *mcp.Client, serverName string, hooks []config.LifecycleHook) error {
	for _, hook := range hooks {
		result, err := client.CallTool(ctx, hook.Tool, hook.Args)
		if err != nil {
			return formatHookError(serverName, hook, err)
		}

		// Check for tool-level error in the result.
		if result != nil && result.IsError {
			msg := "unknown error"
			if len(result.Content) > 0 {
				msg = result.Content[0].Text
			}
			return formatHookError(serverName, hook, fmt.Errorf("%s", msg))
		}
	}
	return nil
}

// formatHookError creates a user-friendly error message for lifecycle hook failures.
func formatHookError(serverName string, hook config.LifecycleHook, err error) error {
	var hint string

	switch hook.Tool {
	case "activate_project":
		project := "<unknown>"
		if p, ok := hook.Args["project"]; ok {
			project = fmt.Sprintf("%v", p)
		}
		hint = fmt.Sprintf(
			"Hint: ensure the project exists at %s and has been onboarded\n"+
				"  Run: mcpx %s onboarding",
			project, serverName,
		)
	default:
		hint = fmt.Sprintf("Hint: check that the MCP server supports tool %q", hook.Tool)
	}

	// Build a multi-line error with context.
	var b strings.Builder
	fmt.Fprintf(&b, "server %q: lifecycle hook %q failed\n", serverName, hook.Tool)
	fmt.Fprintf(&b, "  Error: %s\n", err)
	fmt.Fprintf(&b, "  %s", hint)

	return fmt.Errorf("%s", b.String())
}
