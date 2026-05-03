package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
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
// Prints a one-line URL hint to stderr at most once per shell session.
//
// Suppressed when:
//   - stderr is not a terminal (pipelines)
//   - the invoked command's primary output is data the user wants to read
//     (find / gain / batch / doctor / version / --json globally)
//   - a per-session marker file at ~/.mcpx/.notice-<PPID> exists.
func startUIIfEnabled(cfg *config.Config) {
	uiCfg := (*config.UIConfig)(nil).Default()
	if cfg != nil && cfg.UI != nil {
		uiCfg = cfg.UI.Default()
	}
	enabled := uiCfg.Enabled != nil && *uiCfg.Enabled
	ui.EnsureRunningAsync(enabled, uiCfg.Port, uiCfg.Bind)

	if !enabled {
		return
	}
	if !isStderrTerminal() || noticeAlreadyShown() || isQuietCommand() {
		return
	}
	go func() {
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			h, err := ui.LoadHandshake()
			if err == nil && h.Port > 0 {
				fmt.Fprintf(os.Stderr, "mcpx dashboard: http://%s:%d/?t=%s\n", h.Bind, h.Port, h.Token)
				markNoticeShown()
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
}

// isQuietCommand reports whether the invoked subcommand is one whose primary
// output is structured data — adding a stderr URL banner would visually
// interfere with copy-paste or human reading of the output.
func isQuietCommand() bool {
	if len(os.Args) < 2 {
		return false
	}
	switch os.Args[1] {
	case "find", "gain", "batch", "doctor", "version", "completion", "ui":
		return true
	}
	for _, a := range os.Args {
		if a == "--json" {
			return true
		}
	}
	return false
}

// noticeAlreadyShown returns true if a marker file exists for this shell session
// (keyed by parent PID so a fresh terminal still gets the notice).
func noticeAlreadyShown() bool {
	path := noticeMarkerPath()
	if path == "" {
		return false
	}
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func markNoticeShown() {
	path := noticeMarkerPath()
	if path == "" {
		return
	}
	_ = os.MkdirAll(strings.TrimSuffix(path, "/"+strings.Split(path, "/")[len(strings.Split(path, "/"))-1]), 0o755)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
	if err == nil {
		f.Close()
	}
}

func noticeMarkerPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s/.mcpx/.notice-%d", home, os.Getppid())
}
