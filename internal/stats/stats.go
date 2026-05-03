// Package stats records every mcpx tool invocation as a JSONL line on disk.
// Reads, aggregations, and dashboard queries all consume the same file.
//
// Writes are async and never block or error out the caller. If the buffer
// overflows we drop entries and increment a counter (see DroppedCount).
package stats

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// Record is one mcpx tool invocation. One JSONL line per record.
type Record struct {
	ID                  string         `json:"id,omitempty"`
	TS                  time.Time      `json:"ts"`
	Session             string         `json:"session,omitempty"`
	Project             string         `json:"project,omitempty"`
	Agent               string         `json:"agent,omitempty"`
	Server              string         `json:"server"`
	Tool                string         `json:"tool"`
	Args                map[string]any `json:"args,omitempty"`
	ArgsBytes           int            `json:"args_bytes"`
	ArgsTokensEst       int            `json:"args_tokens_est"`
	ResponseBytes       int            `json:"response_bytes"`
	ResponseTokensEst   int            `json:"response_tokens_est"`
	ResultPreview       string         `json:"result_preview,omitempty"`
	ResultTruncated     bool           `json:"result_truncated,omitempty"`
	LatencyMS           int64          `json:"latency_ms"`
	Transport           string         `json:"transport,omitempty"`
	Daemon              bool           `json:"daemon,omitempty"`
	SchemaCacheHit      bool           `json:"schema_cache_hit,omitempty"`
	ExitCode            int            `json:"exit_code"`
	Error               string         `json:"error,omitempty"`
	PolicyAction        string         `json:"policy_action,omitempty"`
	PolicyName          string         `json:"policy_name,omitempty"`
	NativeBaselineToks  int            `json:"native_baseline_tokens,omitempty"`
	TokensSaved         int            `json:"tokens_saved"`
}

// MaxResultPreviewBytes caps the inlined result preview to keep JSONL rows bounded.
const MaxResultPreviewBytes = 2 * 1024

// NewID returns a short stable identifier for a record. Combines unix-nano with
// a small random suffix; readable in logs and short enough for URLs.
func NewID() string {
	const alpha = "0123456789abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = alpha[int(b[i])%len(alpha)]
	}
	return time.Now().UTC().Format("20060102T150405") + "-" + string(b)
}

// TruncateForPreview returns at most MaxResultPreviewBytes of s and a flag.
func TruncateForPreview(s string) (string, bool) {
	if len(s) <= MaxResultPreviewBytes {
		return s, false
	}
	return s[:MaxResultPreviewBytes], true
}

// Writer appends Records to a JSONL file. Writes are buffered and flushed by
// a background goroutine; the caller never blocks on disk I/O.
type Writer struct {
	path     string
	ch       chan Record
	dropped  atomic.Int64
	closeMu  sync.Mutex
	closed   bool
	doneCh   chan struct{}
	enabled  atomic.Bool
}

// NewWriter creates a Writer that appends to path. Buffer size caps in-flight
// records; on overflow Write() drops the record and bumps DroppedCount.
//
// Pass enabled=false to make every Write a no-op (config: gain.enabled=false).
func NewWriter(path string, bufSize int, enabled bool) (*Writer, error) {
	if bufSize <= 0 {
		bufSize = 1024
	}
	w := &Writer{
		path:   path,
		ch:     make(chan Record, bufSize),
		doneCh: make(chan struct{}),
	}
	w.enabled.Store(enabled)
	if enabled {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		go w.run()
	} else {
		close(w.doneCh)
	}
	return w, nil
}

// Write enqueues a record. Never blocks; never errors. If the buffer is full
// the record is silently dropped and DroppedCount() increments.
func (w *Writer) Write(r Record) {
	if w == nil || !w.enabled.Load() {
		return
	}
	if r.TS.IsZero() {
		r.TS = time.Now().UTC()
	}
	select {
	case w.ch <- r:
	default:
		w.dropped.Add(1)
	}
}

// DroppedCount returns the number of records dropped due to buffer overflow.
func (w *Writer) DroppedCount() int64 {
	if w == nil {
		return 0
	}
	return w.dropped.Load()
}

// Close drains the buffer and stops the writer goroutine. Safe to call once.
func (w *Writer) Close() error {
	if w == nil {
		return nil
	}
	w.closeMu.Lock()
	if w.closed {
		w.closeMu.Unlock()
		return nil
	}
	w.closed = true
	close(w.ch)
	w.closeMu.Unlock()
	<-w.doneCh
	return nil
}

// Flush waits for all currently buffered records to be written. Convenience
// wrapper for tests; prefer Close in production paths.
func (w *Writer) Flush(timeout time.Duration) error {
	if w == nil || !w.enabled.Load() {
		return nil
	}
	deadline := time.After(timeout)
	for {
		if len(w.ch) == 0 {
			return nil
		}
		select {
		case <-deadline:
			return errors.New("stats: flush timeout")
		case <-time.After(2 * time.Millisecond):
		}
	}
}

func (w *Writer) run() {
	defer close(w.doneCh)
	for r := range w.ch {
		w.append(r)
	}
}

func (w *Writer) append(r Record) {
	data, err := json.Marshal(r)
	if err != nil {
		return
	}
	data = append(data, '\n')
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	_, _ = f.Write(data)
	_ = f.Close()
}

// EstimateTokens returns a char-based token estimate (bytes / 4).
// Honest approximation; real tokenizer is opt-in via build tag in v1.6.1.
func EstimateTokens(b []byte) int {
	if len(b) == 0 {
		return 0
	}
	return (len(b) + 3) / 4
}

// EstimateTokensString is EstimateTokens for strings.
func EstimateTokensString(s string) int {
	if s == "" {
		return 0
	}
	return (len(s) + 3) / 4
}

// DefaultPath returns the conventional stats file path under the user home dir.
// Empty string on failure.
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".mcpx", "stats.jsonl")
}
