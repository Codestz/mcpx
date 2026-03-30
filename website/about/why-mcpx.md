# Why MCPX

The Model Context Protocol gave AI systems a standard way to call tools — semantic code search, structured reasoning, databases, web browsing. Hundreds of MCP servers exist today.

But deploying MCP in production has three unsolved problems.

## 1. Context Cost

Loading MCP servers into an AI session is expensive. Each server dumps its full tool schema into the conversation context **upfront**. Five servers? That's 50,000-100,000 tokens consumed before the AI does anything.

| Servers Loaded | Context Cost | Impact |
|---------------|-------------|--------|
| 3 servers | ~30-60K tokens | Noticeable context reduction |
| 5 servers | ~50-100K tokens | Significant. Less room for code. |
| 10 servers | ~100K+ tokens | Context nearly full before work begins |
| **Any count (mcpx)** | **0 tokens** | **Tools called on demand via Bash** |

The AI carries those definitions for the entire session — whether it uses them or not.

**mcpx eliminates this.** Instead of loading schemas, the AI calls `mcpx <server> <tool>` through Bash when it needs a tool. On-demand discovery, zero upfront cost.

## 2. No Security

Every MCP tool call is unrestricted. There's no authorization layer, no policy enforcement, no audit trail.

An AI agent connected to a Postgres MCP can `DROP TABLE` as easily as it can `SELECT`. A code search server can be pointed at files outside the project. A Slack MCP can post to any channel.

**mcpx adds the missing security layer.** Policy enforcement, security modes (read-only, editing), content inspection (SQL mutation blocking), argument validation (path traversal prevention), and a JSONL audit log that records every call.

## 3. Multi-Server Management

Each MCP server needs its own security profile — a database MCP needs different rules than a code search MCP. And when two developers work on different projects simultaneously, their daemons shouldn't interfere.

**mcpx handles this automatically.** Per-server security policies, scoped daemon isolation, and a single config file that manages everything.

## The Insight

AI agents already know how to use terminals. They compose commands, pipe output, parse JSON. The terminal is the universal interface.

mcpx converts every MCP server into a CLI command — and adds the security controls that production teams need.

## What This Unlocks

### Context efficiency
Zero tokens upfront. The AI discovers tools with `mcpx list` and `--help` only when needed.

### Security
Policy enforcement, audit logging, read-only modes. Teams can adopt MCP without giving AI agents unrestricted access.

### Speed
Daemon mode keeps heavy servers warm. Sub-millisecond startup for the CLI itself. `<5ms` for tool calls to a warm daemon.

### Composability
Every MCP tool becomes a UNIX command. Pipe between servers, redirect output, compose with `jq`, `grep`, `xargs`.

### Scalability
Adding a new server doesn't increase context cost. 5 servers or 50 — the AI pays the same: zero tokens upfront.

## What MCPX Is Not

- **Not a replacement for MCP.** mcpx wraps MCP servers — it doesn't replace the protocol.
- **Not an AI client.** It doesn't make decisions. It's infrastructure between the agent and the server.
- **Not only for humans.** The primary user is the AI agent. Humans benefit from the security and audit capabilities.

## Why Go

- **Single binary.** No runtime dependencies. Ship one file.
- **Sub-millisecond startup.** The CLI must be faster than the MCP server.
- **Cross-platform.** macOS, Linux, Windows from one codebase.
- **Concurrency.** Daemon mode, socket handling, and transport management need goroutines.
