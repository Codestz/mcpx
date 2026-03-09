package daemon

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// NewDaemonRunCommand creates the hidden __daemon command that runs the daemon process.
// This is invoked by EnsureRunning as a detached subprocess.
func NewDaemonRunCommand() *cobra.Command {
	var (
		command     string
		argsB64     string
		envB64      string
		idleMinutes int
	)

	cmd := &cobra.Command{
		Use:    "__daemon [server]",
		Short:  "Internal: run daemon for a server",
		Args:   cobra.ExactArgs(1),
		Hidden: true,
		RunE: func(cmd *cobra.Command, cliArgs []string) error {
			serverName := cliArgs[0]

			args, err := decodeStringSlice(argsB64)
			if err != nil {
				return fmt.Errorf("decode args: %w", err)
			}

			env, err := decodeStringSlice(envB64)
			if err != nil {
				return fmt.Errorf("decode env: %w", err)
			}

			idleTimeout := time.Duration(idleMinutes) * time.Minute

			return Start(serverName, command, args, env, idleTimeout)
		},
	}

	cmd.Flags().StringVar(&command, "command", "", "Server command")
	cmd.Flags().StringVar(&argsB64, "args", "", "Server args (base64 JSON)")
	cmd.Flags().StringVar(&envB64, "env", "", "Server env (base64 JSON)")
	cmd.Flags().IntVar(&idleMinutes, "idle", 30, "Idle timeout in minutes")
	cmd.MarkFlagRequired("command")

	return cmd
}

// EncodeStringSlice encodes a string slice as base64 JSON for passing to __daemon.
func EncodeStringSlice(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	data, _ := json.Marshal(ss)
	return base64.StdEncoding.EncodeToString(data)
}

func decodeStringSlice(s string) ([]string, error) {
	if s == "" {
		return nil, nil
	}
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	var result []string
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// NewDaemonManageCommand creates the user-facing "daemon" command with status/stop subcommands.
func NewDaemonManageCommand(serverNames []string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manage server daemons",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show running daemons",
		RunE: func(cmd *cobra.Command, args []string) error {
			found := false
			for _, name := range serverNames {
				if IsRunning(name) {
					fmt.Printf("  %s  running  %s\n", name, SocketPath(name))
					found = true
				}
			}
			if !found {
				fmt.Println("No daemons running.")
			}
			return nil
		},
	})

	stopCmd := &cobra.Command{
		Use:   "stop [server]",
		Short: "Stop a daemon",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return serverNames, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := Stop(args[0]); err != nil {
				return err
			}
			fmt.Printf("Stopped daemon for %s\n", args[0])
			return nil
		},
	}
	cmd.AddCommand(stopCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "stop-all",
		Short: "Stop all running daemons",
		RunE: func(cmd *cobra.Command, args []string) error {
			stopped := 0
			for _, name := range serverNames {
				if IsRunning(name) {
					if err := Stop(name); err != nil {
						fmt.Printf("  %s  error: %v\n", name, err)
					} else {
						fmt.Printf("  %s  stopped\n", name)
						stopped++
					}
				}
			}
			if stopped == 0 {
				fmt.Println("No daemons were running.")
			}
			return nil
		},
	})

	return cmd
}
