package ui

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/codestz/mcpx/internal/stats"
)

//go:embed assets/*
var assetsFS embed.FS

// Run starts the dashboard HTTP server. Blocks until ctx is cancelled or the
// idle timeout fires.
func Run(ctx context.Context, port int, bind string, idleTimeout time.Duration) error {
	if bind == "" {
		bind = "127.0.0.1"
	}
	addr := bind + ":" + strconv.Itoa(port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		// If the requested port is taken, try an ephemeral one.
		listener, err = net.Listen("tcp", bind+":0")
		if err != nil {
			return fmt.Errorf("ui: listen: %w", err)
		}
	}
	defer listener.Close()

	resolved := listener.Addr().(*net.TCPAddr)
	token := NewToken()

	if err := SaveHandshake(Handshake{
		Port: resolved.Port, Token: token, PID: os.Getpid(), Bind: bind,
	}); err != nil {
		return fmt.Errorf("ui: handshake: %w", err)
	}
	defer RemoveHandshake()

	srv := newServer(token, idleTimeout)
	httpSrv := &http.Server{
		Handler:           srv,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Idle watchdog: when no requests in idleTimeout, shutdown gracefully.
	idleCtx, cancelIdle := context.WithCancel(ctx)
	defer cancelIdle()
	go srv.idleLoop(idleCtx, idleTimeout, func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- httpSrv.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

type server struct {
	token       string
	mux         *http.ServeMux
	tmpl        *template.Template
	lastHit     atomic.Int64
	idleTimeout time.Duration

	subsMu sync.Mutex
	subs   map[chan stats.Record]struct{}

	tailMu     sync.Mutex
	statsPath  string
	tailOffset int64
}

func newServer(token string, idleTimeout time.Duration) *server {
	s := &server{
		token:       token,
		idleTimeout: idleTimeout,
		subs:        map[chan stats.Record]struct{}{},
		statsPath:   stats.DefaultPath(),
	}
	s.lastHit.Store(time.Now().UnixNano())

	tmpl, err := template.ParseFS(assetsFS, "assets/index.html")
	if err == nil {
		s.tmpl = tmpl
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/assets/", s.handleAssets)
	mux.HandleFunc("/api/summary", s.tokenGuard(s.handleSummary))
	mux.HandleFunc("/api/projects", s.tokenGuard(s.handleProjects))
	mux.HandleFunc("/api/events", s.tokenGuard(s.handleEvents))
	mux.HandleFunc("/api/call/", s.tokenGuard(s.handleCall))
	mux.HandleFunc("/api/audit", s.tokenGuard(s.handleAudit))
	mux.HandleFunc("/api/tool/", s.tokenGuard(s.handleToolDetail))
	s.mux = mux

	go s.tailLoop()
	return s
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.lastHit.Store(time.Now().UnixNano())
	s.mux.ServeHTTP(w, r)
}

func (s *server) tokenGuard(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("t") != s.token {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		h(w, r)
	}
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Browser hits "/" without token, then loads JS that includes ?t=<token>.
	// Anyone with the URL+token can access — that's the entire auth model.
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	tok := r.URL.Query().Get("t")
	if tok != s.token {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!doctype html><meta charset=utf-8>
<title>mcpx</title>
<body style="font-family:system-ui;padding:40px;color:#c9d1d9;background:#0d1117">
<h2>mcpx dashboard</h2>
<p style="color:#6e7681">Open the URL printed by the CLI on first run, or run:<br>
<code style="background:#161b22;padding:4px 8px;border-radius:4px">mcpx ui open</code></p>
</body>`)
		return
	}
	if s.tmpl == nil {
		http.Error(w, "template missing", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.tmpl.Execute(w, map[string]string{"Token": tok})
}

func (s *server) handleAssets(w http.ResponseWriter, r *http.Request) {
	tok := r.URL.Query().Get("t")
	if tok != s.token {
		http.Error(w, "forbidden", 403)
		return
	}
	rel := strings.TrimPrefix(r.URL.Path, "/assets/")
	data, err := fs.ReadFile(assetsFS, "assets/"+rel)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	switch filepath.Ext(rel) {
	case ".js":
		w.Header().Set("Content-Type", "application/javascript")
	case ".css":
		w.Header().Set("Content-Type", "text/css")
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	}
	_, _ = w.Write(data)
}

func (s *server) handleSummary(w http.ResponseWriter, r *http.Request) {
	since, _ := time.ParseDuration(r.URL.Query().Get("since"))
	if since == 0 {
		since = 7 * 24 * time.Hour
	}
	filter := stats.Filter{Since: time.Now().Add(-since)}

	if r.URL.Query().Get("all") == "" {
		if p := r.URL.Query().Get("project"); p != "" {
			filter.Project = p
		}
	}

	summary, err := stats.Aggregate(s.statsPath, filter, 30)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Per-server call counts for the sidebar.
	perServer := map[string]int{}
	for _, st := range summary.TopServers {
		perServer[st.Server] = st.Calls
	}
	resp := map[string]any{
		"Calls":          summary.Calls,
		"TokensSaved":    summary.TokensSaved,
		"NativeBaseline": summary.NativeBaseline,
		"AvgLatencyMS":   summary.AvgLatencyMS,
		"P50LatencyMS":   summary.P50LatencyMS,
		"P95LatencyMS":   summary.P95LatencyMS,
		"CacheHitRate":   summary.CacheHitRate,
		"SchemaHitRate":  summary.SchemaHitRate,
		"ErrorRate":      summary.ErrorRate,
		"TopTools":       summary.TopTools,
		"TopSavers":      summary.TopSavers,
		"Daily":          summary.Daily,
		"Recent":         summary.Recent,
		"Servers":        summary.Servers,
		"Projects":       summary.Projects,
		"PerServer":      perServer,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// handleProjects lists distinct projects observed in stats with per-project totals.
func (s *server) handleProjects(w http.ResponseWriter, r *http.Request) {
	type proj struct {
		Name        string
		Path        string
		Calls       int
		TokensSaved int64
		Current     bool
	}
	cwd, _ := os.Getwd()

	// Load broad summary across all time.
	summary, err := stats.Aggregate(s.statsPath, stats.Filter{}, 0)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Walk records once to build per-project counts.
	type bucket struct {
		Calls       int
		TokensSaved int64
	}
	by := map[string]*bucket{}
	_ = stats.Iter(s.statsPath, stats.Filter{}, func(rec stats.Record) error {
		if rec.Project == "" {
			return nil
		}
		b, ok := by[rec.Project]
		if !ok {
			b = &bucket{}
			by[rec.Project] = b
		}
		b.Calls++
		b.TokensSaved += int64(rec.TokensSaved)
		return nil
	})
	_ = summary

	out := make([]proj, 0, len(by))
	for path, b := range by {
		out = append(out, proj{
			Name:        filepath.Base(path),
			Path:        path,
			Calls:       b.Calls,
			TokensSaved: b.TokensSaved,
			Current:     path == cwd,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

// handleCall returns the full record for a given call ID.
func (s *server) handleCall(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/call/")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	var found *stats.Record
	_ = stats.Iter(s.statsPath, stats.Filter{}, func(rec stats.Record) error {
		if rec.ID == id {
			cp := rec
			found = &cp
		}
		return nil
	})
	if found == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(found)
}

// handleAudit returns recent denied/warn policy decisions.
func (s *server) handleAudit(w http.ResponseWriter, r *http.Request) {
	limit := 100
	var rows []stats.Record
	_ = stats.Iter(s.statsPath, stats.Filter{}, func(rec stats.Record) error {
		if rec.PolicyAction == "denied" || rec.PolicyAction == "warn" {
			rows = append(rows, rec)
			if len(rows) > limit {
				rows = rows[1:]
			}
		}
		return nil
	})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rows)
}

// handleToolDetail returns aggregated stats and recent calls for one tool.
func (s *server) handleToolDetail(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/tool/")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 {
		http.Error(w, "expected /api/tool/<server>/<tool>", http.StatusBadRequest)
		return
	}
	server, tool := parts[0], parts[1]

	var recent []stats.Record
	calls := 0
	var totalLat, totalSaved int64
	var errors int
	latencies := []int64{}

	_ = stats.Iter(s.statsPath, stats.Filter{Server: server, Tool: tool}, func(rec stats.Record) error {
		calls++
		totalLat += rec.LatencyMS
		totalSaved += int64(rec.TokensSaved)
		latencies = append(latencies, rec.LatencyMS)
		if rec.ExitCode != 0 {
			errors++
		}
		recent = append(recent, rec)
		if len(recent) > 50 {
			recent = recent[1:]
		}
		return nil
	})

	avg := 0.0
	if calls > 0 {
		avg = float64(totalLat) / float64(calls)
	}
	resp := map[string]any{
		"server":      server,
		"tool":        tool,
		"calls":       calls,
		"errors":      errors,
		"tokens_saved": totalSaved,
		"avg_latency_ms": avg,
		"recent":      recent,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// handleEvents streams new stats records over Server-Sent Events.
func (s *server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan stats.Record, 64)
	s.subsMu.Lock()
	s.subs[ch] = struct{}{}
	s.subsMu.Unlock()

	defer func() {
		s.subsMu.Lock()
		delete(s.subs, ch)
		s.subsMu.Unlock()
	}()

	enc := json.NewEncoder(w)
	for {
		select {
		case <-r.Context().Done():
			return
		case rec := <-ch:
			fmt.Fprintf(w, "event: call\ndata: ")
			_ = enc.Encode(rec)
			fmt.Fprintf(w, "\n")
			flusher.Flush()
		}
	}
}

// tailLoop watches the stats JSONL file and broadcasts new records to
// SSE subscribers.
func (s *server) tailLoop() {
	if s.statsPath == "" {
		return
	}
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for range tick.C {
		s.tailOnce()
	}
}

func (s *server) tailOnce() {
	s.tailMu.Lock()
	defer s.tailMu.Unlock()

	f, err := os.Open(s.statsPath)
	if err != nil {
		return
	}
	defer f.Close()
	stat, _ := f.Stat()
	if stat == nil {
		return
	}
	if s.tailOffset == 0 {
		// First poll: skip to end so we only stream NEW records.
		s.tailOffset = stat.Size()
		return
	}
	if stat.Size() < s.tailOffset {
		// File rotated/truncated.
		s.tailOffset = stat.Size()
		return
	}
	if stat.Size() == s.tailOffset {
		return
	}
	if _, err := f.Seek(s.tailOffset, io.SeekStart); err != nil {
		return
	}
	dec := json.NewDecoder(f)
	for dec.More() {
		var rec stats.Record
		if err := dec.Decode(&rec); err != nil {
			break
		}
		s.broadcast(rec)
	}
	s.tailOffset = stat.Size()
}

func (s *server) broadcast(rec stats.Record) {
	s.subsMu.Lock()
	defer s.subsMu.Unlock()
	for ch := range s.subs {
		select {
		case ch <- rec:
		default:
			// Slow subscriber — drop rather than block.
		}
	}
}

// idleLoop shuts the server down after idleTimeout of no activity.
func (s *server) idleLoop(ctx context.Context, idleTimeout time.Duration, shutdown func()) {
	if idleTimeout <= 0 {
		return
	}
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			last := s.lastHit.Load()
			if time.Since(time.Unix(0, last)) > idleTimeout {
				shutdown()
				return
			}
		}
	}
}
