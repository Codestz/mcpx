package cli

import (
	"fmt"
	"os"
	"runtime/debug"
	"sort"

	"github.com/codestz/mcpx/internal/config"
	"github.com/codestz/mcpx/internal/daemon"
	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags.
// Falls back to Go module version from build info (works with go install).
var Version = func() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}()

// globalOpts holds flags shared across all commands.
type globalOpts struct {
	jsonOutput bool
	quiet      bool
	dryRun     bool
	pick       string
	timeout    string
}

func (o *globalOpts) outputMode() outputMode {
	if o.quiet {
		return outputQuiet
	}
	if o.jsonOutput {
		return outputJSON
	}
	return outputPretty
}

// Exit codes.
const (
	exitOK         = 0
	exitToolError  = 1
	exitConfigErr  = 2
	exitConnectErr = 3
)

// Execute is the main entry point for the CLI.
func Execute() {
	opts := &globalOpts{}

	root := &cobra.Command{
		Use:           "mcpx",
		Short:         "MCP server proxy — call MCP tools from the command line",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().BoolVar(&opts.jsonOutput, "json", false, "Output raw JSON")
	root.PersistentFlags().BoolVar(&opts.quiet, "quiet", false, "Suppress output")
	root.PersistentFlags().BoolVar(&opts.dryRun, "dry-run", false, "Show what would execute without running")

	// Static commands.
	root.AddCommand(versionCmd())
	root.AddCommand(listCmd(opts))
	root.AddCommand(initCmd())
	root.AddCommand(configureCmd())
	root.AddCommand(completionCmd())
	root.AddCommand(pingCmd(opts))
	root.AddCommand(secretCmd(opts))

	// Hidden daemon runner command.
	root.AddCommand(daemon.NewDaemonRunCommand())

	// Dynamic server commands from config.
	if err := addServerCommands(root, opts); err != nil {
		out := newOutput(opts.outputMode())
		out.errorf("config: %v", err)
		os.Exit(exitConfigErr)
	}

	if err := root.Execute(); err != nil {
		out := newOutput(opts.outputMode())
		out.errorf("%v", err)
		os.Exit(exitToolError)
	}
}

// versionCmd prints the version.
func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print mcpx version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("mcpx %s\n", Version)
		},
	}
}

// listCmd lists configured servers, or tools for a specific server.
func listCmd(opts *globalOpts) *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
		Use:               "list [server]",
		Short:             "List configured servers or tools for a server",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeServerNames,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := newOutput(opts.outputMode())

			cfg, _, err := config.Load()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}

			if len(args) == 0 {
				// List all servers.
				servers := make(map[string]serverInfo)
				for name, sc := range cfg.Servers {
					servers[name] = serverInfo{
						Command:   sc.Command,
						Transport: sc.Transport,
						Daemon:    sc.Daemon,
					}
				}
				return out.printServers(servers)
			}

			// List tools for a specific server.
			serverName := args[0]
			sc, ok := cfg.Servers[serverName]
			if !ok {
				var names []string
				for name := range cfg.Servers {
					names = append(names, name)
				}
				sort.Strings(names)
				return fmt.Errorf("server %q not found. Available: %s",
					serverName, joinOr(names))
			}

			client, cleanup, err := connectServer(cmd.Context(), serverName, sc)
			if err != nil {
				return err
			}
			defer cleanup()

			tools, err := client.ListTools(cmd.Context())
			if err != nil {
				return fmt.Errorf("list tools: %w", err)
			}

			if verbose {
				return out.printToolsVerbose(serverName, tools)
			}
			return out.printTools(serverName, tools)
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show tools with all their flags")

	return cmd
}

// completionCmd generates shell completion scripts.
func completionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}
	return cmd
}

// addServerCommands reads the config and adds a subcommand for each server.
func addServerCommands(root *cobra.Command, opts *globalOpts) error {
	cfg, _, err := config.Load()
	if err != nil {
		return err
	}

	// Sort server names for deterministic command order.
	names := make([]string, 0, len(cfg.Servers))
	for name := range cfg.Servers {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		root.AddCommand(buildServerCommand(name, cfg.Servers[name], cfg.Security, opts))
	}

	// Daemon management commands.
	root.AddCommand(daemon.NewDaemonManageCommand(names))

	return nil
}

func joinOr(ss []string) string {
	switch len(ss) {
	case 0:
		return ""
	case 1:
		return ss[0]
	default:
		return fmt.Sprintf("%s or %s", joinComma(ss[:len(ss)-1]), ss[len(ss)-1])
	}
}

func joinComma(ss []string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}
