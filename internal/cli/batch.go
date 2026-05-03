package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/codestz/mcpx/internal/config"
	"github.com/codestz/mcpx/internal/mcp"
	"github.com/codestz/mcpx/internal/stats"
	"github.com/spf13/cobra"
)

// BatchEntry is one input line in the NDJSON batch.
type BatchEntry struct {
	ID     string         `json:"id,omitempty"`
	Server string         `json:"server"`
	Tool   string         `json:"tool"`
	Args   map[string]any `json:"args"`
}

// BatchResult is one output line, emitted in the same order as inputs.
type BatchResult struct {
	ID        string          `json:"id"`
	OK        bool            `json:"ok"`
	LatencyMS int64           `json:"latency_ms"`
	Result    *mcp.CallResult `json:"result,omitempty"`
	Error     string          `json:"error,omitempty"`
	ExitCode  int             `json:"exit_code,omitempty"`
}

func batchCmd(opts *globalOpts) *cobra.Command {
	var (
		parallel      bool
		sequential    bool
		maxConcurrent int
		stopOnError   bool
	)

	cmd := &cobra.Command{
		Use:   "batch",
		Short: "Run many tool calls from NDJSON stdin, in parallel by default",
		Long: `Reads NDJSON from stdin (one JSON object per line, with fields server/tool/args/id)
and runs each call. Emits NDJSON to stdout in the same order as inputs.

One MCP client per server is reused across the whole batch — no handshake-per-call.

Examples:
  cat calls.jsonl | mcpx batch
  cat calls.jsonl | mcpx batch --max-concurrent 4
  mcpx find "search" --json | jq -c '...' | mcpx batch`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if sequential {
				parallel = false
			}
			if maxConcurrent <= 0 {
				maxConcurrent = runtime.NumCPU()
			}

			cfg, _, err := config.Load()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}
			initStats(cfg)

			entries, err := readBatch(os.Stdin)
			if err != nil {
				return err
			}

			results := executeBatch(cmd.Context(), entries, cfg, parallel, maxConcurrent, stopOnError)

			enc := json.NewEncoder(os.Stdout)
			for _, r := range results {
				_ = enc.Encode(r)
			}
			// Optional summary to stderr (suppressed in --quiet).
			if !opts.quiet {
				printBatchSummary(results)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&parallel, "parallel", true, "Run calls concurrently (default)")
	cmd.Flags().BoolVar(&sequential, "sequential", false, "Run calls one at a time")
	cmd.Flags().IntVar(&maxConcurrent, "max-concurrent", 0, "Max concurrent calls (0 = NumCPU)")
	cmd.Flags().BoolVar(&stopOnError, "stop-on-error", false, "Abort the batch on first error")
	return cmd
}

// readBatch parses NDJSON entries from r. Blank lines are skipped. IDs are
// auto-assigned when missing so the output stream stays addressable.
func readBatch(r io.Reader) ([]BatchEntry, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	var entries []BatchEntry
	idx := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e BatchEntry
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, fmt.Errorf("batch: line %d: %w", idx+1, err)
		}
		if e.Server == "" || e.Tool == "" {
			return nil, fmt.Errorf("batch: line %d: server and tool are required", idx+1)
		}
		if e.ID == "" {
			e.ID = strconv.Itoa(idx)
		}
		entries = append(entries, e)
		idx++
	}
	return entries, scanner.Err()
}

// executeBatch runs entries against per-server clients. Per-server clients are
// created lazily and closed at the end. Parallel mode bounds concurrency by
// maxConcurrent. Returns results in the original input order.
func executeBatch(ctx context.Context, entries []BatchEntry, cfg *config.Config, parallel bool, maxConcurrent int, stopOnError bool) []BatchResult {
	results := make([]BatchResult, len(entries))
	clients := newClientPool(cfg)
	defer clients.closeAll()

	type job struct{ idx int }

	exec := func(j job) {
		e := entries[j.idx]
		results[j.idx] = runBatchEntry(ctx, e, clients)
	}

	if !parallel {
		for i := range entries {
			exec(job{i})
			if stopOnError && !results[i].OK {
				// Mark remaining as skipped.
				for k := i + 1; k < len(entries); k++ {
					results[k] = BatchResult{ID: entries[k].ID, OK: false, Error: "skipped (stop-on-error)"}
				}
				return results
			}
		}
		return results
	}

	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	for i := range entries {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			exec(job{i})
		}(i)
	}
	wg.Wait()
	return results
}

func runBatchEntry(ctx context.Context, e BatchEntry, pool *clientPool) BatchResult {
	start := time.Now()
	res := BatchResult{ID: e.ID}

	defer func() {
		preview, truncated := "", false
		if res.Result != nil {
			preview, truncated = stats.TruncateForPreview(extractText(res.Result))
		}
		recordStats(stats.Record{
			ID:              stats.NewID(),
			Server:          e.Server,
			Tool:            e.Tool,
			Args:            e.Args,
			ArgsBytes:       jsonBytes(e.Args),
			ResponseBytes:   jsonBytes(res.Result),
			ResultPreview:   preview,
			ResultTruncated: truncated,
			LatencyMS:       time.Since(start).Milliseconds(),
			ExitCode:        res.ExitCode,
			Error:           res.Error,
			Daemon:          pool.daemon(e.Server),
			Transport:       pool.transport(e.Server),
			PolicyAction:    "allowed",
		})
	}()

	if _, err := pool.findTool(ctx, e.Server, e.Tool); err != nil {
		res.Error = err.Error()
		res.ExitCode = exitToolNotFound
		res.LatencyMS = time.Since(start).Milliseconds()
		return res
	}

	client, err := pool.client(ctx, e.Server)
	if err != nil {
		res.Error = err.Error()
		res.ExitCode = exitConnectErr
		res.LatencyMS = time.Since(start).Milliseconds()
		return res
	}

	result, err := client.CallTool(ctx, e.Tool, e.Args)
	res.LatencyMS = time.Since(start).Milliseconds()
	if err != nil {
		res.Error = err.Error()
		res.ExitCode = exitToolError
		return res
	}
	res.OK = true
	res.Result = result
	return res
}

// printBatchSummary writes an end-of-batch human summary to stderr.
func printBatchSummary(results []BatchResult) {
	ok, fail := 0, 0
	var totalMS int64
	for _, r := range results {
		if r.OK {
			ok++
		} else {
			fail++
		}
		totalMS += r.LatencyMS
	}
	fmt.Fprintf(os.Stderr, "\nbatch: %d ok, %d fail, total %dms (sum)\n", ok, fail, totalMS)
}

// clientPool manages one MCP client per server name across the batch lifetime.
// Lazy connect on first use, reused for subsequent calls, closed on closeAll.
type clientPool struct {
	cfg     *config.Config
	mu      sync.Mutex
	clients map[string]*pooledClient
}

type pooledClient struct {
	c      *mcp.Client
	clean  func()
	tools  []mcp.Tool
	toolMu sync.Mutex
}

func newClientPool(cfg *config.Config) *clientPool {
	return &clientPool{cfg: cfg, clients: map[string]*pooledClient{}}
}

func (p *clientPool) client(ctx context.Context, server string) (*mcp.Client, error) {
	p.mu.Lock()
	pc, ok := p.clients[server]
	p.mu.Unlock()
	if ok {
		return pc.c, nil
	}

	sc, ok := p.cfg.Servers[server]
	if !ok {
		return nil, fmt.Errorf("server %q not in config", server)
	}
	c, clean, err := connectServer(ctx, server, sc)
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	p.clients[server] = &pooledClient{c: c, clean: clean}
	p.mu.Unlock()
	return c, nil
}

func (p *clientPool) findTool(ctx context.Context, server, name string) (*mcp.Tool, error) {
	c, err := p.client(ctx, server)
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	pc := p.clients[server]
	p.mu.Unlock()

	pc.toolMu.Lock()
	defer pc.toolMu.Unlock()
	if pc.tools == nil {
		sc := p.cfg.Servers[server]
		tools, _, _, terr := listToolsCached(ctx, server, sc, c)
		if terr != nil {
			return nil, terr
		}
		pc.tools = tools
	}
	for i := range pc.tools {
		if pc.tools[i].Name == name {
			return &pc.tools[i], nil
		}
	}
	return nil, fmt.Errorf("tool %q not in server %q", name, server)
}

func (p *clientPool) daemon(server string) bool {
	if sc, ok := p.cfg.Servers[server]; ok {
		return sc.Daemon
	}
	return false
}

func (p *clientPool) transport(server string) string {
	if sc, ok := p.cfg.Servers[server]; ok {
		return sc.Transport
	}
	return ""
}

func (p *clientPool) closeAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, pc := range p.clients {
		if pc.clean != nil {
			pc.clean()
		}
	}
}

