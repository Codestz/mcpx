// Package schemacache stores MCP server schemas (initialize result + tools/
// prompts/resources lists) in ~/.mcpx/cache/<server-hash>.json with a TTL.
//
// Hits skip the full MCP handshake on repeat invocations. The native_baseline_tokens
// field — what loading this server natively would cost an agent — is computed
// once at populate time and consumed by the stats writer.
package schemacache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/codestz/mcpx/internal/mcp"
)

// Entry is the on-disk schema cache record for one server.
type Entry struct {
	Version             int                  `json:"version"`
	CapturedAt          time.Time            `json:"captured_at"`
	TTL                 time.Duration        `json:"ttl"`
	ServerHash          string               `json:"server_hash"`
	Initialize          mcp.InitializeResult `json:"initialize"`
	Tools               []mcp.Tool           `json:"tools"`
	Prompts             []mcp.Prompt         `json:"prompts,omitempty"`
	Resources           []mcp.Resource       `json:"resources,omitempty"`
	NativeBaselineToks  int                  `json:"native_baseline_tokens"`
}

// Key derives a stable cache key from server-defining attributes. Different
// command/args/env/url → different cache entry, so config changes are never
// served stale data.
func Key(command string, args []string, env map[string]string, url string) string {
	h := sha256.New()
	_, _ = h.Write([]byte(command))
	_, _ = h.Write([]byte{0})
	for _, a := range args {
		_, _ = h.Write([]byte(a))
		_, _ = h.Write([]byte{0})
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		_, _ = h.Write([]byte(k))
		_, _ = h.Write([]byte("="))
		_, _ = h.Write([]byte(env[k]))
		_, _ = h.Write([]byte{0})
	}
	_, _ = h.Write([]byte(url))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// Path returns the on-disk path for a cache entry. Caller must ensure the
// parent directory exists; Save() handles creation.
func Path(key string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".mcpx", "cache", "schemas", key+".json")
}

// Load reads an entry from disk. Returns (entry, true, nil) on a fresh hit,
// (nil, false, nil) on a miss or expired entry, and an error only on I/O
// failures other than "not exist".
func Load(key string) (*Entry, bool, error) {
	path := Path(key)
	if path == "" {
		return nil, false, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	var e Entry
	if err := json.Unmarshal(data, &e); err != nil {
		// Corrupt cache entry — treat as miss so the caller refetches.
		return nil, false, nil
	}
	if e.TTL > 0 && time.Since(e.CapturedAt) > e.TTL {
		return nil, false, nil
	}
	return &e, true, nil
}

// Save writes the entry to disk. Creates parent directories as needed.
func Save(key string, e *Entry) error {
	path := Path(key)
	if path == "" {
		return errors.New("schemacache: no home dir")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if e.CapturedAt.IsZero() {
		e.CapturedAt = time.Now().UTC()
	}
	if e.Version == 0 {
		e.Version = 1
	}
	if e.ServerHash == "" {
		e.ServerHash = key
	}
	if e.NativeBaselineToks == 0 {
		e.NativeBaselineToks = ComputeBaseline(e)
	}
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Clear removes a single cache entry. No-op if the file doesn't exist.
func Clear(key string) error {
	path := Path(key)
	if path == "" {
		return nil
	}
	err := os.Remove(path)
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}

// ClearAll wipes the entire schema cache.
func ClearAll() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(home, ".mcpx", "cache", "schemas"))
}

// ComputeBaseline approximates how many tokens a native MCP client would burn
// loading this server: stringify init result + tools list + prompts list +
// resources list, then divide by 4 (char heuristic).
func ComputeBaseline(e *Entry) int {
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetIndent("", "  ")
	_ = enc.Encode(e.Initialize)
	_ = enc.Encode(e.Tools)
	_ = enc.Encode(e.Prompts)
	_ = enc.Encode(e.Resources)
	return (b.Len() + 3) / 4
}

// Bypass returns true when callers should skip the cache entirely. Honors the
// MCPX_CACHE=off env var.
func Bypass() bool {
	v := strings.ToLower(os.Getenv("MCPX_CACHE"))
	return v == "off" || v == "0" || v == "false"
}
