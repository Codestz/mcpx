package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/codestz/mcpx/internal/config"
	"github.com/codestz/mcpx/internal/mcp"
	"github.com/codestz/mcpx/internal/schemacache"
)

// listToolsCached returns a server's tools from the schema cache when fresh,
// otherwise calls the live MCP client and persists the response.
//
// Returns the tools, whether the result came from cache, the precomputed
// native_baseline_tokens for this server, and any error from the live call.
func listToolsCached(ctx context.Context, serverName string, sc *config.ServerConfig, client *mcp.Client) ([]mcp.Tool, bool, int, error) {
	if schemacache.Bypass() {
		return fetchAndStore(ctx, serverName, sc, client)
	}

	resolvedArgs, resolvedEnv, _ := resolveServerConfig(sc)
	key := schemacache.Key(sc.Command, resolvedArgs, resolvedEnv, sc.URL)

	entry, hit, _ := schemacache.Load(key)
	if hit {
		return entry.Tools, true, entry.NativeBaselineToks, nil
	}

	tools, err := client.ListTools(ctx)
	if err != nil {
		return nil, false, 0, err
	}
	storeEntry(key, sc, client, tools)
	if e, _, _ := schemacache.Load(key); e != nil {
		return tools, false, e.NativeBaselineToks, nil
	}
	return tools, false, 0, nil
}

func fetchAndStore(ctx context.Context, serverName string, sc *config.ServerConfig, client *mcp.Client) ([]mcp.Tool, bool, int, error) {
	tools, err := client.ListTools(ctx)
	if err != nil {
		return nil, false, 0, err
	}
	resolvedArgs, resolvedEnv, _ := resolveServerConfig(sc)
	key := schemacache.Key(sc.Command, resolvedArgs, resolvedEnv, sc.URL)
	storeEntry(key, sc, client, tools)
	if e, _, _ := schemacache.Load(key); e != nil {
		return tools, false, e.NativeBaselineToks, nil
	}
	return tools, false, 0, nil
}

func storeEntry(key string, sc *config.ServerConfig, client *mcp.Client, tools []mcp.Tool) {
	ttl := schemaTTL(sc)
	caps := client.ServerCapabilities()
	entry := &schemacache.Entry{
		TTL: ttl,
		Initialize: mcp.InitializeResult{
			ProtocolVersion: client.ProtocolVersion(),
			Capabilities:    caps,
			ServerInfo:      client.ServerInfo(),
		},
		Tools: tools,
	}
	// Best-effort prompts/resources fetch — failures don't block.
	if caps.Prompts != nil {
		if p, err := client.ListPrompts(context.Background()); err == nil {
			entry.Prompts = p
		}
	}
	if caps.Resources != nil {
		if r, err := client.ListResources(context.Background()); err == nil {
			entry.Resources = r
		}
	}
	_ = schemacache.Save(key, entry)
}

// schemaTTL resolves the schema cache TTL for a server. Defaults to 5m.
func schemaTTL(_ *config.ServerConfig) time.Duration {
	cfg, _, err := config.Load()
	if err != nil || cfg == nil {
		return 5 * time.Minute
	}
	cc := cfg.Cache.Default()
	d, err := time.ParseDuration(cc.SchemaTTL)
	if err != nil {
		return 5 * time.Minute
	}
	return d
}
// callToolCached wraps client.CallTool with the idempotent result cache.
//
// Idempotence is decided per (config, tool) — see schemacache.IsIdempotent.
// Errors and non-idempotent calls always go to the live MCP server. Successful
// idempotent calls are persisted with the configured TTL.
//
// File mtime participates in the cache key when args contain a `relative_path`
// (or `path`/`file`/`file_path`) that resolves to an existing file or directory.
// This keeps cached results from going stale across edits within the TTL window:
// any change to the file shifts the key and the next call falls through to a
// fresh fetch.
func callToolCached(ctx context.Context, client *mcp.Client, serverName string, tool *mcp.Tool, args map[string]any) (*mcp.CallResult, bool, error) {
	cc, allow, heur, ttl, enabled := resolveResultCachePolicy()
	useCache := enabled && !schemacache.Bypass() && cc != nil &&
		schemacache.IsIdempotent(tool, serverName, allow, heur)

	if useCache {
		root, _ := findProjectRoot()
		extras := fileMtimeExtras(args, root)
		key := schemacache.ResultKey(serverName, tool.Name, args, extras...)
		if entry, hit, _ := schemacache.LoadResult(key); hit {
			r := entry.Result
			return &r, true, nil
		}
		result, err := client.CallTool(ctx, tool.Name, args)
		if err != nil || result == nil || result.IsError {
			return result, false, err
		}
		_ = schemacache.SaveResult(key, &schemacache.ResultEntry{
			TTL:    ttl,
			Server: serverName,
			Tool:   tool.Name,
			Result: *result,
		})
		return result, false, nil
	}

	result, err := client.CallTool(ctx, tool.Name, args)
	return result, false, err
}

// fileMtimeExtras inspects args for keys that conventionally hold filesystem
// paths (`relative_path`, `path`, `file`, `file_path`) and returns
// `path:mtime-ns` strings for each one that resolves to an existing file or
// directory under root.
//
// Edits change the mtime, which changes the cache key, which forces a fresh
// fetch. Tools that read entire subtrees (e.g. recursive search) can still go
// stale if a deep file changes without touching the named path —
// `MCPX_CACHE=off` is the right escape hatch for those cases.
func fileMtimeExtras(args map[string]any, root string) []string {
	if len(args) == 0 {
		return nil
	}
	candidates := []string{"relative_path", "path", "file", "file_path"}

	var out []string
	for _, key := range candidates {
		raw, ok := args[key]
		if !ok {
			continue
		}
		p, ok := raw.(string)
		if !ok || p == "" {
			continue
		}
		full := p
		if !filepath.IsAbs(p) && root != "" {
			full = filepath.Join(root, p)
		}
		info, err := os.Stat(full)
		if err != nil {
			continue
		}
		out = append(out, fmt.Sprintf("%s:%d", p, info.ModTime().UnixNano()))
	}
	return out
}

func resolveResultCachePolicy() (*config.CacheConfig, []string, bool, time.Duration, bool) {
	cfg, _, err := config.Load()
	if err != nil || cfg == nil {
		return nil, nil, false, 0, false
	}
	cc := cfg.Cache.Default()
	enabled := cc.ResultEnabled != nil && *cc.ResultEnabled
	if !enabled {
		return &cc, nil, false, 0, false
	}
	d, err := time.ParseDuration(cc.ResultTTL)
	if err != nil {
		d = 30 * time.Second
	}
	return &cc, cc.ResultIdempotent, cc.ResultHeuristic, d, true
}
