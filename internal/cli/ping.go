package cli

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/codestz/mcpx/internal/config"
	"github.com/spf13/cobra"
)

// pingCmd creates the "ping" command for health-checking an MCP server.
func pingCmd(opts *globalOpts) *cobra.Command {
	return &cobra.Command{
		Use:               "ping <server>",
		Short:             "Check if an MCP server is alive and responding",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeServerNames,
		Run: func(cmd *cobra.Command, args []string) {
			out := newOutput(opts.outputMode())
			serverName := args[0]

			cfg, _, err := config.Load()
			if err != nil {
				out.errorf("config: %v", err)
				os.Exit(exitConfigErr)
			}

			sc, ok := cfg.Servers[serverName]
			if !ok {
				var names []string
				for name := range cfg.Servers {
					names = append(names, name)
				}
				sort.Strings(names)
				out.errorf("server %q not found. Available: %s", serverName, joinOr(names))
				os.Exit(exitConfigErr)
			}

			start := time.Now()
			client, cleanup, err := connectServer(cmd.Context(), serverName, sc)
			if err != nil {
				out.errorf("%v", err)
				os.Exit(exitConnectErr)
			}
			defer cleanup()

			tools, err := client.ListTools(cmd.Context())
			if err != nil {
				out.errorf("list tools: %v", err)
				os.Exit(exitConnectErr)
			}
			elapsed := time.Since(start)

			if err := out.printPing(serverName, len(tools), elapsed); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(exitToolError)
			}
		},
	}
}
