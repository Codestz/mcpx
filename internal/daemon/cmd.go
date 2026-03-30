package daemon

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// NewDaemonRunCommand creates the hidden __daemon command that runs the daemon process.
// This is invoked by EnsureRunning as a detached subprocess.
func NewDaemonRunCommand() *cobra.Command {
	var (
		command     string
		scope       string
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

			return Start(serverName, scope, command, args, env, idleTimeout)
		},
	}

	cmd.Flags().StringVar(&command, "command", "", "Server command")
	cmd.Flags().StringVar(&scope, "scope", "", "Daemon scope (project hash)")
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

// runningDaemon holds info about a discovered daemon.
type runningDaemon struct {
	serverName string
	scope      string
	socketPath string
}

// discoverRunning finds all running daemons for the given server names
// by globbing PID files in /tmp.
func discoverRunning(serverNames []string) []runningDaemon {
	uid := os.Getuid()
	var found []runningDaemon

	for _, name := range serverNames {
		// Match both scoped and unscoped PID files.
		pattern := fmt.Sprintf("/tmp/mcpx-%s-*-%d.pid", name, uid)
		matches, _ := filepath.Glob(pattern)

		// Also check the unscoped path.
		unscopedPID := fmt.Sprintf("/tmp/mcpx-%s-%d.pid", name, uid)
		if _, err := os.Stat(unscopedPID); err == nil {
			// Check it's not already in matches (unscoped matches the glob too
			// when there's no scope — the uid part looks like a scope).
			// Deduplicate by checking if this PID file is already matched.
			alreadyFound := false
			for _, m := range matches {
				if m == unscopedPID {
					alreadyFound = true
					break
				}
			}
			if !alreadyFound {
				matches = append(matches, unscopedPID)
			}
		}

		for _, pidPath := range matches {
			scope := extractScope(pidPath, name, uid)
			if IsRunning(name, scope) {
				found = append(found, runningDaemon{
					serverName: name,
					scope:      scope,
					socketPath: SocketPath(name, scope),
				})
			}
		}
	}

	return found
}

// extractScope extracts the scope from a PID file path.
// /tmp/mcpx-serena-a1b2c3d4-501.pid → "a1b2c3d4"
// /tmp/mcpx-serena-501.pid → ""
func extractScope(pidPath, serverName string, uid int) string {
	// Expected patterns:
	// scoped:   /tmp/mcpx-{server}-{scope}-{uid}.pid
	// unscoped: /tmp/mcpx-{server}-{uid}.pid
	prefix := fmt.Sprintf("/tmp/mcpx-%s-", serverName)
	suffix := fmt.Sprintf("-%d.pid", uid)

	if !strings.HasPrefix(pidPath, prefix) || !strings.HasSuffix(pidPath, suffix) {
		return ""
	}

	middle := pidPath[len(prefix) : len(pidPath)-len(suffix)]

	// If middle is empty, it's unscoped (the uid was directly after server name).
	// But we need to check: /tmp/mcpx-serena-501.pid has middle="" when suffix is "-501.pid"
	// Actually, let's just check if the path equals the unscoped path.
	unscopedPath := fmt.Sprintf("/tmp/mcpx-%s-%d.pid", serverName, uid)
	if pidPath == unscopedPath {
		return ""
	}

	// Verify middle looks like a scope (hex string).
	if len(middle) > 0 {
		// Check it's not just the uid (already handled above).
		if _, err := strconv.Atoi(middle); err != nil {
			return middle
		}
	}

	return ""
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
			daemons := discoverRunning(serverNames)
			if len(daemons) == 0 {
				fmt.Println("No daemons running.")
				return nil
			}
			for _, d := range daemons {
				label := d.serverName
				if d.scope != "" {
					label += " (" + d.scope + ")"
				}
				fmt.Printf("  %s  running  %s\n", label, d.socketPath)
			}
			return nil
		},
	})

	stopCmd := &cobra.Command{
		Use:   "stop [server]",
		Short: "Stop a daemon (stops all scoped instances of the server)",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return serverNames, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			daemons := discoverRunning([]string{name})
			if len(daemons) == 0 {
				return fmt.Errorf("daemon: %s not running", name)
			}
			for _, d := range daemons {
				if err := Stop(d.serverName, d.scope); err != nil {
					fmt.Printf("  %s (%s)  error: %v\n", d.serverName, d.scope, err)
				} else {
					label := d.serverName
					if d.scope != "" {
						label += " (" + d.scope + ")"
					}
					fmt.Printf("  Stopped %s\n", label)
				}
			}
			return nil
		},
	}
	cmd.AddCommand(stopCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "stop-all",
		Short: "Stop all running daemons",
		RunE: func(cmd *cobra.Command, args []string) error {
			daemons := discoverRunning(serverNames)
			if len(daemons) == 0 {
				fmt.Println("No daemons were running.")
				return nil
			}
			for _, d := range daemons {
				if err := Stop(d.serverName, d.scope); err != nil {
					fmt.Printf("  %s  error: %v\n", d.serverName, err)
				} else {
					label := d.serverName
					if d.scope != "" {
						label += " (" + d.scope + ")"
					}
					fmt.Printf("  %s  stopped\n", label)
				}
			}
			return nil
		},
	})

	return cmd
}
