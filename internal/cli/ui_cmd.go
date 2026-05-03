package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/codestz/mcpx/internal/config"
	"github.com/codestz/mcpx/internal/ui"
	"github.com/spf13/cobra"
)

// uiRunCmd is the hidden background entrypoint spawned by ui.EnsureRunningAsync.
// Equivalent in spirit to the existing __daemon command.
func uiRunCmd() *cobra.Command {
	var port int
	var bind string
	var idle time.Duration

	cmd := &cobra.Command{
		Use:    "__ui-run",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
			go func() {
				<-sigCh
				cancel()
			}()
			return ui.Run(ctx, port, bind, idle)
		},
	}
	cmd.Flags().IntVar(&port, "port", 7878, "TCP port (0=ephemeral)")
	cmd.Flags().StringVar(&bind, "bind", "127.0.0.1", "Bind address")
	cmd.Flags().DurationVar(&idle, "idle-timeout", 1*time.Hour, "Idle shutdown")
	return cmd
}

// uiManageCmd exposes user-facing dashboard controls: status, stop, open, disable.
func uiManageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Manage the always-on dashboard daemon",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show dashboard URL or 'inactive'",
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := ui.LoadHandshake()
			if err != nil {
				fmt.Println("inactive")
				return nil
			}
			fmt.Printf("http://%s:%d/?t=%s  (pid %d)\n", h.Bind, h.Port, h.Token, h.PID)
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "stop",
		Short: "Stop the dashboard daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := ui.LoadHandshake()
			if err != nil {
				fmt.Println("not running")
				return nil
			}
			if h.PID > 0 {
				if proc, err := os.FindProcess(h.PID); err == nil {
					_ = proc.Signal(syscall.SIGTERM)
				}
			}
			_ = ui.RemoveHandshake()
			fmt.Println("stopped")
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "open",
		Short: "Print the dashboard URL (suitable for `open $(...)`)",
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := ui.LoadHandshake()
			if err != nil {
				return fmt.Errorf("dashboard not running")
			}
			fmt.Printf("http://%s:%d/?t=%s\n", h.Bind, h.Port, h.Token)
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "disable",
		Short: "Hint how to disable the dashboard permanently",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Add to ~/.mcpx/config.yml:")
			fmt.Println()
			fmt.Println("  ui:")
			fmt.Println("    enabled: false")
			fmt.Println()
			fmt.Println("Or set MCPX_UI=off in your environment.")
		},
	})
	return cmd
}

// startUIIfEnabled spawns the dashboard daemon according to config + env.
//
// The dashboard runs silently — its URL is surfaced only when the user asks:
//   - `mcpx gain`        prints the URL in its footer when the daemon is up
//   - `mcpx ui status`   prints the URL or "inactive"
//   - `mcpx ui open`     prints the URL alone (use with `open $(...)`)
//
// No CLI invocation prints the banner unsolicited. Opt out of spawning entirely
// with `MCPX_UI=off` or `ui.enabled: false` in config.
func startUIIfEnabled(cfg *config.Config) {
	uiCfg := (*config.UIConfig)(nil).Default()
	if cfg != nil && cfg.UI != nil {
		uiCfg = cfg.UI.Default()
	}
	enabled := uiCfg.Enabled != nil && *uiCfg.Enabled
	ui.EnsureRunningAsync(enabled, uiCfg.Port, uiCfg.Bind)
}

