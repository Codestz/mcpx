# Comparison

How mcpx compares to other approaches for giving AI agents access to MCP tools.

## MCPX vs Native MCP

| Aspect | Native MCP | MCPX |
|--------|-----------|------|
| **Context cost** | 10-30K tokens per server, upfront | 0 tokens upfront |
| **Discovery** | All schemas loaded at session start | On-demand via `mcpx list` |
| **Startup** | Every session pays init cost | Daemon mode: init once |
| **Max servers** | ~5-10 before context fills | Unlimited |
| **Composability** | Server-to-server: not possible | Pipe stdout between servers |
| **Integration** | Client-specific (Claude Code, Cursor) | Any tool that runs Bash |

**Choose native MCP when:**
- You use 1-2 small servers with few tools
- You need bidirectional communication (server-initiated notifications)
- Your AI client doesn't support Bash execution

**Choose mcpx when:**
- You use 3+ servers or servers with many tools
- Context efficiency matters
- You want tools to work across AI clients
- You need server-to-server composition

## MCPX vs REST API Wrappers

Some people wrap MCP servers behind REST APIs and call them via `curl`.

| Aspect | REST Wrapper | MCPX |
|--------|-------------|------|
| **Setup** | Write HTTP server + deploy | `mcpx init` |
| **Discovery** | Custom (Swagger/OpenAPI) | Built-in (`--help`, `list`) |
| **Auth** | You implement it | OS keychain |
| **Process management** | You manage it (systemd, Docker) | Daemon mode built in |
| **Schema mapping** | Manual | Automatic from MCP |

mcpx is the REST wrapper approach without the REST wrapper.

## MCPX vs Direct Subprocess Calls

You could have the AI spawn MCP servers directly via `exec.Command` equivalent:

| Aspect | Direct Subprocess | MCPX |
|--------|-------------------|------|
| **Handshake** | AI must implement MCP init | mcpx handles it |
| **Schema parsing** | AI parses JSON schema | Flags auto-generated |
| **Connection reuse** | New process every call | Daemon mode |
| **Error handling** | AI parses JSON-RPC errors | Exit codes + stderr |
| **Config** | Hardcoded in AI instructions | YAML config files |

## Context Cost Breakdown

Real-world measurements of schema sizes:

| Server | Tools | Schema Size | Native MCP Cost |
|--------|-------|-------------|----------------|
| Serena | 21 | ~18K chars | ~20K tokens |
| Sequential Thinking | 1 | ~2K chars | ~3K tokens |
| Filesystem | 11 | ~8K chars | ~10K tokens |
| Brave Search | 2 | ~3K chars | ~4K tokens |
| GitHub | 30+ | ~25K chars | ~30K tokens |

With mcpx, all of these cost 0 tokens until the AI actually needs them. A single `mcpx list serena -v` call costs ~500 tokens in the response — and only when needed.
