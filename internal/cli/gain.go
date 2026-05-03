package cli

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/codestz/mcpx/internal/config"
	"github.com/codestz/mcpx/internal/render"
	"github.com/codestz/mcpx/internal/stats"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func gainCmd(opts *globalOpts) *cobra.Command {
	var (
		allProjects bool
		project     string
		since       time.Duration
		byMode      string
		history     int
		watch       bool
		suggest     bool
	)

	cmd := &cobra.Command{
		Use:   "gain",
		Short: "Show mcpx token-savings dashboard",
		Long: `Renders a premium terminal dashboard summarizing every mcpx call:
tokens saved vs native MCP loading, top tools, cache hit rate, recent calls.

Examples:
  mcpx gain                       # current project, last 7 days
  mcpx gain --all                 # every project
  mcpx gain --since 24h
  mcpx gain --by tool             # ranked tools only
  mcpx gain --history 20
  mcpx gain --suggest             # mined recommendations
  mcpx gain --watch               # live refresh
  mcpx gain --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if since == 0 {
				since = 7 * 24 * time.Hour
			}
			runGain := func() error {
				return renderGainOnce(opts, runGainArgs{
					all: allProjects, project: project, since: since,
					byMode: byMode, history: history, suggest: suggest,
				})
			}

			if watch {
				return watchLoop(runGain)
			}
			return runGain()
		},
	}
	cmd.Flags().BoolVar(&allProjects, "all", false, "Aggregate across every project")
	cmd.Flags().StringVar(&project, "project", "", "Filter to a specific project root")
	cmd.Flags().DurationVar(&since, "since", 0, "Time window (default 7 days)")
	cmd.Flags().StringVar(&byMode, "by", "", "Group by tool|server|day")
	cmd.Flags().IntVar(&history, "history", 0, "Show last N calls only")
	cmd.Flags().BoolVar(&watch, "watch", false, "Live refresh every second")
	cmd.Flags().BoolVar(&suggest, "suggest", false, "Mine the data for recommendations")
	return cmd
}

type runGainArgs struct {
	all     bool
	project string
	since   time.Duration
	byMode  string
	history int
	suggest bool
}

func renderGainOnce(opts *globalOpts, a runGainArgs) error {
	cfg, projectRoot, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	gainCfg := (*config.GainConfig)(nil).Default()
	if cfg != nil && cfg.Gain != nil {
		gainCfg = cfg.Gain.Default()
	}
	path := gainCfg.StatsPath
	if path == "" {
		path = stats.DefaultPath()
	}

	filter := stats.Filter{Since: time.Now().Add(-a.since)}
	if !a.all {
		if a.project != "" {
			filter.Project = a.project
		} else if projectRoot != "" {
			filter.Project = projectRoot
		}
	}

	summary, err := stats.Aggregate(path, filter, max(a.history, 5))
	if err != nil {
		return err
	}

	if opts.outputMode() == outputJSON {
		out := newOutput(outputJSON)
		return out.printJSON(summary)
	}

	switch a.byMode {
	case "tool":
		renderToolTable(summary)
		return nil
	case "server":
		renderServerTable(summary)
		return nil
	case "day":
		renderDayTable(summary)
		return nil
	}
	if a.history > 0 {
		renderHistory(summary, a.history)
		return nil
	}
	if a.suggest {
		renderSuggestions(summary)
		return nil
	}

	dashURL := readDashURL()
	scope := "this project · " + humanizeDuration(a.since)
	if a.all {
		scope = "all projects · " + humanizeDuration(a.since)
	} else if a.project != "" {
		scope = "project " + truncProject(a.project) + " · " + humanizeDuration(a.since)
	}
	renderGainDashboard(summary, scope, dashURL)
	return nil
}

func renderGainDashboard(s *stats.Summary, scope, dashURL string) {
	w := render.TermWidth()
	if w > 100 {
		w = 100
	}
	if w < 70 {
		w = 70
	}

	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen, color.Bold)
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed)
	dim := color.New(color.FgHiBlack)

	hr := func() {
		fmt.Println("  " + dim.Sprint(strings.Repeat(render.BoxH, w-4)))
	}

	// Header
	fmt.Println()
	cyan.Print("  mcpx ")
	dim.Printf("  %s", scope)
	saved := "saved: " + render.FormatNumber(s.TokensSaved)
	pad := w - 8 - len(scope) - len(saved)
	if pad < 1 {
		pad = 1
	}
	fmt.Print(strings.Repeat(" ", pad))
	green.Println(saved)
	hr()

	// Hero metric
	fmt.Println()
	bold.Print("  Tokens saved   ")
	green.Printf("%s", render.FormatNumber(s.TokensSaved))
	dim.Printf("   (vs %s native)\n", render.FormatNumber(s.NativeBaseline))

	if len(s.Daily) > 0 {
		var vals []int64
		for _, d := range s.Daily {
			vals = append(vals, d.TokensSaved)
		}
		// Pad to constant width when fewer days exist than the window.
		fmt.Print("                 ")
		dim.Println(render.Sparkline(vals) + "  daily")
	}
	fmt.Println()

	// Stat grid (2 rows x 3 cols)
	col1 := 24
	col2 := 24
	col3 := 24
	row := func(k1, v1, k2, v2, k3, v3 string) {
		fmt.Print("  ")
		dim.Printf("%-12s", k1)
		bold.Printf("%-*s", col1-12, v1)
		dim.Printf("%-12s", k2)
		bold.Printf("%-*s", col2-12, v2)
		dim.Printf("%-12s", k3)
		bold.Printf("%-*s", col3-12, v3)
		fmt.Println()
	}
	row(
		"Calls", render.FormatNumber(int64(s.Calls)),
		"Cache hit", render.FormatPercent(s.CacheHitRate),
		"Errors", colorErr(red, s.ErrorRate),
	)
	row(
		"Avg latency", render.FormatDuration(int64(s.AvgLatencyMS)),
		"p95", render.FormatDuration(s.P95LatencyMS),
		"Servers", fmt.Sprintf("%d", len(s.Servers)),
	)
	fmt.Println()

	// Top tools (by calls)
	if len(s.TopTools) > 0 {
		hr()
		bold.Println("  Top tools (by calls)")
		fmt.Println()
		topTools := s.TopTools
		if len(topTools) > 5 {
			topTools = topTools[:5]
		}
		maxCalls := topTools[0].Calls
		nameW := tableNameWidth(topTools, w-30)
		for _, t := range topTools {
			full := t.Server + "." + t.Tool
			full = render.Truncate(full, nameW)
			pct := float64(t.Calls) / float64(maxCalls)
			bar := render.Bar(pct, 18)
			fmt.Print("  ")
			yellow.Printf("%-*s", nameW, full)
			fmt.Printf("  %5d  ", t.Calls)
			green.Println(bar)
		}
		fmt.Println()
	}

	// Top savings
	if hasSavings(s.TopSavers) {
		hr()
		bold.Println("  Top savings (vs native MCP)")
		fmt.Println()
		topSavers := s.TopSavers
		if len(topSavers) > 5 {
			topSavers = topSavers[:5]
		}
		nameW := tableNameWidth(topSavers, w-30)
		for _, t := range topSavers {
			if t.TokensSaved <= 0 {
				continue
			}
			full := render.Truncate(t.Server+"."+t.Tool, nameW)
			fmt.Print("  ")
			yellow.Printf("%-*s", nameW, full)
			green.Printf("  %8s", render.FormatNumber(t.TokensSaved))
			dim.Printf("  saved\n")
		}
		fmt.Println()
	}

	// Recent
	if len(s.Recent) > 0 {
		hr()
		bold.Println("  Last calls")
		fmt.Println()
		for i := len(s.Recent) - 1; i >= 0; i-- {
			r := s.Recent[i]
			tstamp := r.TS.Local().Format("15:04")
			full := render.Truncate(r.Server+"."+r.Tool, 36)
			fmt.Print("  ")
			dim.Printf("%s  ", tstamp)
			fmt.Printf("%-36s  ", full)
			latency := render.FormatDuration(r.LatencyMS)
			if r.SchemaCacheHit {
				yellow.Printf("%6s  warm  ", latency)
			} else {
				fmt.Printf("%6s        ", latency)
			}
			if r.ExitCode != 0 {
				red.Println("  ✗")
			} else {
				green.Println("  ✓")
			}
		}
		fmt.Println()
	}

	// Footer
	hr()
	if dashURL != "" {
		dim.Print("  dashboard ")
		fmt.Println(dashURL)
	} else {
		dim.Println("  dashboard inactive (start mcpx-ui or run any tool call)")
	}
	dim.Println("  token counts are estimates (bytes ÷ 4 — close approximation of Claude tokenization)")
	fmt.Println()
}

func renderToolTable(s *stats.Summary) {
	bold := color.New(color.Bold)
	dim := color.New(color.FgHiBlack)
	yellow := color.New(color.FgYellow)
	green := color.New(color.FgGreen)

	bold.Println("\n  Tools")
	fmt.Println()
	if len(s.TopTools) == 0 {
		dim.Println("  (no calls in window)")
		return
	}
	max := s.TopTools[0].Calls
	for _, t := range s.TopTools {
		full := render.Truncate(t.Server+"."+t.Tool, 36)
		bar := render.Bar(float64(t.Calls)/float64(max), 18)
		fmt.Print("  ")
		yellow.Printf("%-36s", full)
		fmt.Printf("  %5d  ", t.Calls)
		green.Print(bar)
		dim.Printf("  %s saved  %s avg\n",
			render.FormatNumber(t.TokensSaved),
			render.FormatDuration(int64(t.AvgLatencyMS)),
		)
	}
	fmt.Println()
}

func renderServerTable(s *stats.Summary) {
	bold := color.New(color.Bold)
	dim := color.New(color.FgHiBlack)
	yellow := color.New(color.FgYellow)
	green := color.New(color.FgGreen)

	bold.Println("\n  Servers")
	fmt.Println()
	for _, ss := range s.TopServers {
		fmt.Print("  ")
		yellow.Printf("%-24s", ss.Server)
		fmt.Printf("  %5d calls  ", ss.Calls)
		green.Printf("%s saved  ", render.FormatNumber(ss.TokensSaved))
		dim.Printf("%s avg\n", render.FormatDuration(int64(ss.AvgLatencyMS)))
	}
	fmt.Println()
}

func renderDayTable(s *stats.Summary) {
	bold := color.New(color.Bold)
	dim := color.New(color.FgHiBlack)
	green := color.New(color.FgGreen)

	bold.Println("\n  Daily")
	fmt.Println()
	if len(s.Daily) == 0 {
		dim.Println("  (no calls)")
		return
	}
	var max int64
	for _, d := range s.Daily {
		if d.TokensSaved > max {
			max = d.TokensSaved
		}
	}
	for _, d := range s.Daily {
		bar := render.Bar(float64(d.TokensSaved)/float64(maxOr1(max)), 24)
		fmt.Print("  ")
		dim.Print(d.Day.Format("2006-01-02"))
		fmt.Printf("  %5d  ", d.Calls)
		green.Print(bar)
		dim.Printf("  %s saved\n", render.FormatNumber(d.TokensSaved))
	}
	fmt.Println()
}

func renderHistory(s *stats.Summary, n int) {
	bold := color.New(color.Bold)
	dim := color.New(color.FgHiBlack)
	yellow := color.New(color.FgYellow)
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)

	bold.Printf("\n  Last %d calls\n", n)
	fmt.Println()
	if len(s.Recent) == 0 {
		dim.Println("  (no calls)")
		return
	}
	for i := len(s.Recent) - 1; i >= 0; i-- {
		r := s.Recent[i]
		fmt.Print("  ")
		dim.Print(r.TS.Local().Format("Mon 15:04:05"))
		fmt.Print("  ")
		yellow.Printf("%-36s", render.Truncate(r.Server+"."+r.Tool, 36))
		latency := render.FormatDuration(r.LatencyMS)
		fmt.Printf("  %6s", latency)
		if r.SchemaCacheHit {
			yellow.Print("  warm")
		}
		if r.ExitCode != 0 {
			red.Println("  ✗")
		} else {
			green.Println("  ✓")
		}
	}
	fmt.Println()
}

func renderSuggestions(s *stats.Summary) {
	bold := color.New(color.Bold)
	dim := color.New(color.FgHiBlack)
	yellow := color.New(color.FgYellow)

	bold.Println("\n  Recommendations")
	fmt.Println()
	suggestions := mineSuggestions(s)
	if len(suggestions) == 0 {
		dim.Println("  No actionable patterns yet — run more calls and try again.")
		return
	}
	for _, sg := range suggestions {
		fmt.Print("  ")
		yellow.Print("• ")
		fmt.Println(sg)
	}
	fmt.Println()
}

// mineSuggestions returns terse, actionable recommendations from a Summary.
// Heuristics deliberately conservative — false positives erode trust.
func mineSuggestions(s *stats.Summary) []string {
	var out []string
	if s.Calls < 5 {
		return nil
	}
	if s.SchemaHitRate < 0.4 {
		out = append(out, fmt.Sprintf("Schema cache hit rate %s — verify cache.schema_ttl is reasonable.",
			render.FormatPercent(s.SchemaHitRate)))
	}
	if s.ErrorRate > 0.1 {
		out = append(out, fmt.Sprintf("Error rate %s — top failing tools may need argument fixes.",
			render.FormatPercent(s.ErrorRate)))
	}
	if s.P95LatencyMS > 2000 {
		out = append(out, fmt.Sprintf("p95 latency %s — slow tools dominate; consider mcpx batch for parallel execution.",
			render.FormatDuration(s.P95LatencyMS)))
	}
	if len(s.TopTools) > 0 {
		t := s.TopTools[0]
		share := float64(t.Calls) / float64(s.Calls)
		if share > 0.5 {
			out = append(out, fmt.Sprintf("%s.%s accounts for %s of all calls — strong candidate for an alias or shortcut.",
				t.Server, t.Tool, render.FormatPercent(share)))
		}
	}
	return out
}

func tableNameWidth(items any, max int) int {
	w := 24
	switch v := items.(type) {
	case []stats.ToolStat:
		for _, t := range v {
			full := t.Server + "." + t.Tool
			if len(full) > w {
				w = len(full)
			}
		}
	}
	if w > max {
		w = max
	}
	return w
}

func hasSavings(ts []stats.ToolStat) bool {
	for _, t := range ts {
		if t.TokensSaved > 0 {
			return true
		}
	}
	return false
}

func maxOr1(n int64) int64 {
	if n <= 0 {
		return 1
	}
	return n
}

func colorErr(red *color.Color, rate float64) string {
	if rate > 0.05 {
		return red.Sprint(render.FormatPercent(rate))
	}
	return render.FormatPercent(rate)
}

func humanizeDuration(d time.Duration) string {
	if d == 0 {
		return "all time"
	}
	if d >= 24*time.Hour {
		days := int(d / (24 * time.Hour))
		return fmt.Sprintf("%dd", days)
	}
	if d >= time.Hour {
		return fmt.Sprintf("%dh", int(d/time.Hour))
	}
	return fmt.Sprintf("%dm", int(d/time.Minute))
}

func truncProject(p string) string {
	if len(p) <= 32 {
		return p
	}
	return "…" + p[len(p)-30:]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// readDashURL returns the dashboard URL from ~/.mcpx/ui.json if the daemon is alive.
func readDashURL() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	uiInfo, err := readUIHandshake(home)
	if err != nil || uiInfo.Port == 0 {
		return ""
	}
	addr := net.JoinHostPort(uiInfo.Bind, fmt.Sprintf("%d", uiInfo.Port))
	conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
	if err != nil {
		return ""
	}
	conn.Close()
	return fmt.Sprintf("http://%s/?t=%s", addr, uiInfo.Token)
}

// watchLoop redraws the dashboard once per second until interrupted.
func watchLoop(run func() error) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		fmt.Print("\033[H\033[2J") // clear screen
		if err := run(); err != nil {
			return err
		}
		<-ticker.C
	}
}
