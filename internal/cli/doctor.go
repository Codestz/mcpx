package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/codestz/mcpx/internal/config"
	"github.com/codestz/mcpx/internal/daemon"
	"github.com/codestz/mcpx/internal/resolver"
	"github.com/codestz/mcpx/internal/secret"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type checkLevel int

const (
	checkOK checkLevel = iota
	checkWarn
	checkFail
)

type checkResult struct {
	Server string
	Name   string
	Level  checkLevel
	Detail string
}

func doctorCmd(opts *globalOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose mcpx configuration and connectivity",
		Long: `Runs a series of checks against the mcpx config and every configured server.
Verifies syntax, command paths, transports, secret resolution, and daemons.

Examples:
  mcpx doctor              # human table
  mcpx doctor --json       # machine-readable`,
		RunE: func(cmd *cobra.Command, args []string) error {
			results := runDoctor(cmd.Context())
			if opts.outputMode() == outputJSON {
				out := newOutput(outputJSON)
				return out.printJSON(results)
			}
			return printDoctor(results)
		},
	}
	return cmd
}

// runDoctor performs every check and returns the result list.
func runDoctor(ctx context.Context) []checkResult {
	var results []checkResult

	cfg, projectRoot, err := config.Load()
	if err != nil {
		return []checkResult{{Name: "config", Level: checkFail, Detail: err.Error()}}
	}
	results = append(results, checkResult{
		Name: "config", Level: checkOK,
		Detail: fmt.Sprintf("loaded %d server(s)%s", len(cfg.Servers), projectScope(projectRoot)),
	})

	if cfg.Servers == nil || len(cfg.Servers) == 0 {
		results = append(results, checkResult{
			Name: "servers", Level: checkWarn,
			Detail: "no servers configured — add to .mcpx/config.yml or ~/.mcpx/config.yml",
		})
		return results
	}

	res := resolver.New(projectRoot, secret.NewKeyringStore())

	for name, sc := range cfg.Servers {
		results = append(results, checkServer(ctx, name, sc, res)...)
	}

	return results
}

func checkServer(ctx context.Context, name string, sc *config.ServerConfig, res *resolver.Resolver) []checkResult {
	var out []checkResult

	switch sc.Transport {
	case "stdio", "":
		if sc.Command == "" {
			out = append(out, checkResult{
				Server: name, Name: "command", Level: checkFail,
				Detail: "missing required `command` field",
			})
			return out
		}
		path, err := exec.LookPath(sc.Command)
		if err != nil {
			out = append(out, checkResult{
				Server: name, Name: "command", Level: checkFail,
				Detail: fmt.Sprintf("%q not on $PATH", sc.Command),
			})
		} else {
			out = append(out, checkResult{
				Server: name, Name: "command", Level: checkOK,
				Detail: path,
			})
		}
	case "http", "sse":
		url, err := res.Resolve(sc.URL)
		if err != nil {
			out = append(out, checkResult{
				Server: name, Name: "url", Level: checkFail,
				Detail: fmt.Sprintf("could not resolve url: %v", err),
			})
			return out
		}
		client := &http.Client{Timeout: 3 * time.Second}
		req, _ := http.NewRequestWithContext(ctx, "HEAD", url, nil)
		resp, err := client.Do(req)
		if err != nil {
			out = append(out, checkResult{
				Server: name, Name: "url", Level: checkFail,
				Detail: fmt.Sprintf("HEAD %s: %v", url, err),
			})
		} else {
			resp.Body.Close()
			out = append(out, checkResult{
				Server: name, Name: "url", Level: checkOK,
				Detail: fmt.Sprintf("%s — %d", url, resp.StatusCode),
			})
		}
	}

	// Check secret resolution for auth + headers (without leaking values).
	for _, ref := range collectSecretRefs(sc) {
		if _, err := res.Resolve(ref); err != nil {
			out = append(out, checkResult{
				Server: name, Name: "secret", Level: checkFail,
				Detail: fmt.Sprintf("%s: %v", ref, err),
			})
		} else {
			out = append(out, checkResult{
				Server: name, Name: "secret", Level: checkOK,
				Detail: ref + " resolved",
			})
		}
	}

	// Daemon liveness when daemon=true.
	if sc.Daemon {
		scope := daemonScope()
		if daemon.IsRunning(name, scope) {
			out = append(out, checkResult{
				Server: name, Name: "daemon", Level: checkOK,
				Detail: "running, socket reachable",
			})
		} else {
			out = append(out, checkResult{
				Server: name, Name: "daemon", Level: checkWarn,
				Detail: "no daemon yet — first call will spawn one",
			})
		}
	}

	// Live initialize check (3s budget).
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	client, cleanup, err := connectServer(cctx, name, sc)
	if err != nil {
		out = append(out, checkResult{
			Server: name, Name: "initialize", Level: checkFail,
			Detail: shortErr(err),
		})
		return out
	}
	defer cleanup()
	out = append(out, checkResult{
		Server: name, Name: "initialize", Level: checkOK,
		Detail: client.ServerInfo().Name + " " + client.ServerInfo().Version,
	})

	tools, err := client.ListTools(cctx)
	if err != nil {
		out = append(out, checkResult{
			Server: name, Name: "tools/list", Level: checkWarn,
			Detail: shortErr(err),
		})
	} else {
		out = append(out, checkResult{
			Server: name, Name: "tools/list", Level: checkOK,
			Detail: fmt.Sprintf("%d tool(s)", len(tools)),
		})
	}

	return out
}

// collectSecretRefs walks a server config for $(secret.*) references.
func collectSecretRefs(sc *config.ServerConfig) []string {
	var refs []string
	scan := func(s string) {
		if strings.Contains(s, "$(secret.") {
			refs = append(refs, s)
		}
	}
	if sc.Auth != nil {
		scan(sc.Auth.Token)
	}
	for _, v := range sc.Headers {
		scan(v)
	}
	return refs
}

func projectScope(root string) string {
	if root == "" {
		return ", global only"
	}
	return ", project = " + root
}

func shortErr(err error) string {
	s := err.Error()
	if len(s) > 80 {
		return s[:77] + "…"
	}
	return s
}

func printDoctor(results []checkResult) error {
	bold := color.New(color.Bold)
	dim := color.New(color.FgHiBlack)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed)

	fmt.Println()
	bold.Println("  mcpx doctor")
	dim.Println("  ─────────────────────────────────────────────────────")

	currServer := "(global)"
	for _, r := range results {
		srv := r.Server
		if srv == "" {
			srv = "(global)"
		}
		if srv != currServer {
			fmt.Println()
			bold.Print("  " + srv + "\n")
			currServer = srv
		}
		switch r.Level {
		case checkOK:
			green.Print("    [ok]   ")
		case checkWarn:
			yellow.Print("    [warn] ")
		case checkFail:
			red.Print("    [fail] ")
		}
		fmt.Printf("%-14s ", r.Name)
		dim.Println(r.Detail)
	}

	fails := 0
	warns := 0
	for _, r := range results {
		if r.Level == checkFail {
			fails++
		} else if r.Level == checkWarn {
			warns++
		}
	}
	fmt.Println()
	dim.Println("  ─────────────────────────────────────────────────────")
	if fails > 0 {
		red.Printf("  %d failure(s)", fails)
		if warns > 0 {
			fmt.Printf(", ")
			yellow.Printf("%d warning(s)", warns)
		}
		fmt.Println()
		os.Exit(2)
	} else if warns > 0 {
		yellow.Printf("  %d warning(s)\n", warns)
	} else {
		green.Println("  all checks passed")
	}
	fmt.Println()
	return nil
}

