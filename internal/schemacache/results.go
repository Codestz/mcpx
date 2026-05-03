package schemacache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/codestz/mcpx/internal/mcp"
)

// ResultEntry is a cached tool call response. Only successful, idempotent
// calls are cached. Errors and non-idempotent calls bypass the cache entirely.
type ResultEntry struct {
	CapturedAt time.Time      `json:"captured_at"`
	TTL        time.Duration  `json:"ttl"`
	Server     string         `json:"server"`
	Tool       string         `json:"tool"`
	ArgsHash   string         `json:"args_hash"`
	Result     mcp.CallResult `json:"result"`
}

// ResultKey hashes (server, tool, normalized-args-JSON) into a stable cache key.
// Args are normalized via canonical JSON so map ordering doesn't break cache hits.
func ResultKey(server, tool string, args map[string]any) string {
	canon := canonicalize(args)
	h := sha256.New()
	_, _ = h.Write([]byte(server))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(tool))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(canon))
	return hex.EncodeToString(h.Sum(nil))[:24]
}

// resultPath returns the on-disk path for a cached result.
func resultPath(key string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".mcpx", "cache", "results", key+".json")
}

// LoadResult reads a cached result. Returns (nil, false, nil) on miss or expiry.
func LoadResult(key string) (*ResultEntry, bool, error) {
	path := resultPath(key)
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
	var e ResultEntry
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, false, nil
	}
	if e.TTL > 0 && time.Since(e.CapturedAt) > e.TTL {
		return nil, false, nil
	}
	return &e, true, nil
}

// SaveResult persists a successful tool result.
func SaveResult(key string, e *ResultEntry) error {
	path := resultPath(key)
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if e.CapturedAt.IsZero() {
		e.CapturedAt = time.Now().UTC()
	}
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ClearResults wipes all cached results.
func ClearResults() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(home, ".mcpx", "cache", "results"))
}

// IsIdempotent decides whether a tool's result may be cached.
//
// Order of evaluation:
//  1. Server-supplied annotation (readOnlyHint) — preserved in tool's Ext.
//  2. Explicit allow-list "server.tool" — exact match.
//  3. Heuristic prefix match (get_/list_/find_/search_/read_) — opt-in.
func IsIdempotent(tool *mcp.Tool, server string, allowList []string, heuristic bool) bool {
	if tool == nil {
		return false
	}
	for _, entry := range allowList {
		if entry == server+"."+tool.Name {
			return true
		}
	}
	if heuristic {
		switch {
		case strings.HasPrefix(tool.Name, "get_"),
			strings.HasPrefix(tool.Name, "list_"),
			strings.HasPrefix(tool.Name, "find_"),
			strings.HasPrefix(tool.Name, "search_"),
			strings.HasPrefix(tool.Name, "read_"):
			return true
		}
	}
	return false
}

// canonicalize produces stable JSON: object keys sorted, nested objects
// sorted recursively. Required for cache keys to match across map iterations.
func canonicalize(v any) string {
	switch x := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var b strings.Builder
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			kj, _ := json.Marshal(k)
			b.Write(kj)
			b.WriteByte(':')
			b.WriteString(canonicalize(x[k]))
		}
		b.WriteByte('}')
		return b.String()
	case []any:
		var b strings.Builder
		b.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(canonicalize(e))
		}
		b.WriteByte(']')
		return b.String()
	default:
		j, _ := json.Marshal(v)
		return string(j)
	}
}
