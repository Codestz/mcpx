package cli

import (
	"context"
	"sort"
	"time"

	"github.com/codestz/mcpx/internal/config"
	"github.com/spf13/cobra"
)

// completeServerNames returns a completion function that suggests configured server names.
func completeServerNames(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	cfg, _, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	names := make([]string, 0, len(cfg.Servers))
	for name := range cfg.Servers {
		names = append(names, name)
	}
	sort.Strings(names)

	return names, cobra.ShellCompDirectiveNoFileComp
}

// completeToolNames returns a completion function that suggests tool names for a server.
// It connects to the server to discover tools (requires the server to be running for daemon servers).
func completeToolNames(serverName string, sc *config.ServerConfig) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		client, cleanup, err := connectServer(ctx, serverName, sc)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		defer cleanup()

		tools, err := client.ListTools(ctx)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		// Add subcommands before tool names.
		names := []string{"info", "prompt", "resource"}

		for _, t := range tools {
			names = append(names, t.Name)
		}

		return names, cobra.ShellCompDirectiveNoFileComp
	}
}
