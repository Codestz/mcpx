package cli

import (
	"context"
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
