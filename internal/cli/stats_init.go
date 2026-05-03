package cli

import (
	"encoding/json"
	"os"
	"strconv"
	"sync"

	"github.com/codestz/mcpx/internal/config"
	"github.com/codestz/mcpx/internal/stats"
)

var (
	statsWriter   *stats.Writer
	statsOnce     sync.Once
	statsAgent    string
	statsSession  string
)

// initStats lazily creates the package-level stats writer using the merged config.
// Safe to call from multiple goroutines; only the first call has effect.
func initStats(cfg *config.Config) {
	statsOnce.Do(func() {
		gainCfg := (*config.GainConfig)(nil).Default()
		if cfg != nil && cfg.Gain != nil {
			gainCfg = cfg.Gain.Default()
		}
		enabled := gainCfg.Enabled != nil && *gainCfg.Enabled
		path := gainCfg.StatsPath
		if path == "" {
			path = stats.DefaultPath()
		}
		w, err := stats.NewWriter(path, 1024, enabled)
		if err == nil {
			statsWriter = w
		}

		// Resolve agent identity once.
		statsAgent = os.Getenv("MCPX_AGENT")
		if statsAgent == "" {
			statsAgent = "unknown"
		}

		// Session = parent PID. Cheap, stable per-shell, no PII.
		statsSession = strconv.Itoa(os.Getppid())
	})
}

// closeStats drains the writer. Called at the end of cli.Execute.
func closeStats() {
	if statsWriter != nil {
		_ = statsWriter.Close()
	}
}

// recordStats emits a stats record. Safe to call when stats are disabled.
func recordStats(r stats.Record) {
	if statsWriter == nil {
		return
	}
	if r.Agent == "" {
		r.Agent = statsAgent
	}
	if r.Session == "" {
		r.Session = statsSession
	}
	statsWriter.Write(r)
}

// jsonBytes returns the byte length of v marshalled to JSON. 0 on error or nil.
func jsonBytes(v any) int {
	if v == nil {
		return 0
	}
	b, err := json.Marshal(v)
	if err != nil {
		return 0
	}
	return len(b)
}
