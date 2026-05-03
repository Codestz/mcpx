package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/codestz/mcpx/internal/config"
	"github.com/codestz/mcpx/internal/find"
	"github.com/codestz/mcpx/internal/stats"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func findCmd(opts *globalOpts) *cobra.Command {
	var topK int
	var serverFilter string

	cmd := &cobra.Command{
		Use:   "find <query>",
		Short: "Rank MCP tools across all configured servers by relevance",
		Long: `Find ranks tools across every configured MCP server by BM25 relevance to a free-text query.
Built for AI agents at discovery time: instead of scanning 'mcpx list -v' (5–15K tokens),
'mcpx find "search code"' returns 3 ranked candidates in ~80 tokens.

Examples:
  mcpx find "search code by regex"
  mcpx find "github issue" --top 3
  mcpx find "..." --json
  mcpx find "..." --server serena`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			start := time.Now()

			cfg, _, err := config.Load()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}

			servers := cfg.Servers
			if serverFilter != "" {
				if _, ok := cfg.Servers[serverFilter]; !ok {
					return fmt.Errorf("server %q not in config", serverFilter)
				}
				servers = map[string]*config.ServerConfig{serverFilter: cfg.Servers[serverFilter]}
			}

			corpus, perServerHits := gatherCorpus(cmd.Context(), servers)
			results := find.Rank(query, corpus, topK)

			recordStats(stats.Record{
				ID:           stats.NewID(),
				Server:       "mcpx",
				Tool:         "find",
				Args:         map[string]any{"query": query, "top": topK},
				ArgsBytes:    jsonBytes(map[string]any{"query": query, "top": topK}),
				ResponseBytes: jsonBytes(results),
				LatencyMS:    time.Since(start).Milliseconds(),
				ExitCode:     0,
				PolicyAction: "allowed",
			})

			out := newOutput(opts.outputMode())
			if opts.outputMode() == outputJSON {
				return out.printJSON(results)
			}
			printFindResults(query, results, perServerHits)
			return nil
		},
	}
	cmd.Flags().IntVar(&topK, "top", 5, "Max number of results to show")
	cmd.Flags().StringVar(&serverFilter, "server", "", "Restrict search to one server")
	return cmd
}

// gatherCorpus connects to every server, lists its tools (using the schema
// cache), and returns a flat corpus suitable for the BM25 ranker. Errors are
// swallowed per server — a broken server should not break find for others.
// Returns a map of server → tool count for the result-footer.
func gatherCorpus(ctx context.Context, servers map[string]*config.ServerConfig) ([]find.Tool, map[string]int) {
	type result struct {
		name  string
		tools []find.Tool
	}

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results []result
		hits    = map[string]int{}
	)

	for name, sc := range servers {
		wg.Add(1)
		go func(name string, sc *config.ServerConfig) {
			defer wg.Done()
			cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			client, cleanup, err := connectServer(cctx, name, sc)
			if err != nil {
				return
			}
			defer cleanup()

			tools, _, _, err := listToolsCached(cctx, name, sc, client)
			if err != nil {
				return
			}

			out := make([]find.Tool, len(tools))
			for i, t := range tools {
				out[i] = find.Tool{Server: name, Name: t.Name, Description: t.Description}
			}

			mu.Lock()
			results = append(results, result{name: name, tools: out})
			hits[name] = len(tools)
			mu.Unlock()
		}(name, sc)
	}
	wg.Wait()

	var corpus []find.Tool
	for _, r := range results {
		corpus = append(corpus, r.tools...)
	}
	return corpus, hits
}

func printFindResults(query string, results []find.Result, hits map[string]int) {
	bold := color.New(color.Bold)
	dim := color.New(color.FgHiBlack)
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)

	if len(results) == 0 {
		fmt.Fprintln(os.Stderr, "No tools matched.")
		fmt.Fprintf(os.Stderr, "Searched %d server(s): %s\n", len(hits), summarizeServers(hits))
		return
	}

	dim.Fprintf(os.Stdout, "query  ")
	bold.Fprintf(os.Stdout, "%q\n", query)
	dim.Fprintf(os.Stdout, "found  %d result(s) across %d server(s)\n\n", len(results), len(hits))

	maxName := 0
	for _, r := range results {
		full := r.Server + "." + r.Name
		if len(full) > maxName {
			maxName = len(full)
		}
	}
	if maxName > 40 {
		maxName = 40
	}

	for i, r := range results {
		full := r.Server + "." + r.Name
		if len(full) > maxName {
			full = full[:maxName-1] + "…"
		}
		score := fmt.Sprintf("%.2f", normalizeScore(r.Score, results[0].Score))

		switch i {
		case 0:
			cyan.Fprintf(os.Stdout, "  %-*s", maxName, full)
		default:
			bold.Fprintf(os.Stdout, "  %-*s", maxName, full)
		}
		green.Fprintf(os.Stdout, "  %s", score)
		desc := r.Description
		if desc != "" {
			dim.Fprintf(os.Stdout, "  %s", truncate(desc, 60))
		}
		fmt.Fprintln(os.Stdout)
	}

	fmt.Fprintln(os.Stdout)
	dim.Fprintf(os.Stdout, "Run: mcpx <server> <tool> --help for any of the above.\n")
}

func summarizeServers(hits map[string]int) string {
	if len(hits) == 0 {
		return "(none)"
	}
	names := make([]string, 0, len(hits))
	for n := range hits {
		names = append(names, n)
	}
	return strings.Join(names, ", ")
}

// normalizeScore rescales scores to [0,1] relative to the top result.
func normalizeScore(s, top float64) float64 {
	if top == 0 {
		return 0
	}
	v := s / top
	if v > 1 {
		return 1
	}
	if v < 0 {
		return 0
	}
	return v
}
