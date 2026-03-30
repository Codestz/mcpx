```
                         __  __  ____ ____  __  __
                        |  \/  |/ ___|  _ \ \ \/ /
                        | |\/| | |   | |_) | \  /
                        | |  | | |___|  __/  /  \
                        |_|  |_|\____|_|    /_/\_\

          Secure gateway for MCP servers — from CLI to production.
```

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/codestz/mcpx)](https://goreportcard.com/report/github.com/codestz/mcpx)
[![GitHub Downloads](https://img.shields.io/github/downloads/codestz/mcpx/total?logo=github&label=downloads)](https://github.com/codestz/mcpx/releases)
[![Go Downloads](https://pkg.go.dev/badge/github.com/codestz/mcpx.svg)](https://pkg.go.dev/github.com/codestz/mcpx)
[![Homebrew](https://img.shields.io/badge/Homebrew-available-FBB040?logo=homebrew&logoColor=white)](https://github.com/codestz/homebrew-tap)

**[Documentation](https://codestz.github.io/mcpx)** | **[Examples](https://github.com/codestz/mcpx-examples)** | **[Installation](https://codestz.github.io/mcpx/getting-started/installation)** | **[GitHub](https://github.com/codestz/mcpx)**

---

**mcpx** is the control plane between AI agents and [MCP](https://modelcontextprotocol.io) servers. It wraps any MCP server into a CLI command with security policies, audit logging, and scoped daemon isolation — so teams can adopt MCP in production with confidence.

```bash
# Discover → Call → Compose
mcpx serena find_symbol --name "Auth"
mcpx serena search_for_pattern --substring_pattern "TODO" | jq -r '.file'
mcpx postgres query --sql "SELECT * FROM users LIMIT 10"
```

## Why mcpx

MCP servers have three problems in production:

### 1. Context cost

Loading MCP servers natively costs 50-100K tokens per session — before any work starts. mcpx calls tools on demand via Bash. Zero upfront cost.

| Setup | Context Cost | Impact |
|-------|-------------|--------|
| 5 MCP servers (native) | ~80K tokens | Context nearly full |
| **Any count (mcpx)** | **0 tokens** | **Full context available** |

### 2. No security

Every MCP tool call is unrestricted. An AI agent connected to a Postgres MCP can `DROP TABLE` as easily as `SELECT`. There's no authorization, no policy enforcement, no audit trail.

mcpx adds the missing security layer:

```yaml
# Block SQL mutations on your database MCP
security:
  policies:
    - name: no-mutations
      match:
        tools: [query]
        content:
          target: args.sql
          deny_pattern: "(?i)\\b(INSERT|UPDATE|DELETE|DROP|TRUNCATE)\\b"
      action: deny
      message: "Mutation queries blocked"
```

```bash
$ mcpx postgres query --sql "DROP TABLE users"
error: server "postgres": policy "no-mutations" denied tool "query"
  Reason: Mutation queries blocked
```

### 3. Multi-server management

Each server needs its own security profile. A code search MCP needs different rules than a database MCP or a messaging MCP. And when two developers work on different projects simultaneously, their daemons shouldn't interfere.

mcpx handles this with per-server security and scoped daemons:

```yaml
servers:
  serena:
    security:
      mode: editing
      policies:
        - name: source-only
          match:
            args:
              relative_path: { allow_prefix: ["src/", "internal/"] }
          action: deny
  postgres:
    security:
      mode: read-only
```

## Installation

```bash
# Homebrew (macOS / Linux)
brew tap codestz/tap
brew install mcpx

# Go install (requires Go 1.24+)
go install github.com/codestz/mcpx/cmd/mcpx@latest

# Or build from source
git clone https://github.com/codestz/mcpx.git
cd mcpx && make build
```

## Quick Start

### 1. Import existing config

```bash
mcpx init
# Detected .mcp.json — imported 3 servers: serena, sequential-thinking, filesystem
```

### 2. Or configure manually

```yaml
# .mcpx/config.yml
servers:
  serena:
    command: serena
    args: [start-mcp-server, --context=claude-code]
    daemon: true
    startup_timeout: 30s
```

### 3. Use it

```bash
mcpx serena find_symbol --name "Auth"         # call a tool
mcpx ping serena                               # health check
mcpx list serena -v                            # list all tools with flags
```

## Security

mcpx provides three layers of security between AI agents and MCP servers:

### Policies

Define rules that evaluate before every tool call. Supports tool name globs, argument inspection, and content regex:

```yaml
security:
  enabled: true
  global:
    policies:
      - name: no-path-traversal
        match:
          args:
            "*path*": { deny_pattern: "\\.\\.\\/"}
        action: deny
        message: "Path traversal blocked"

servers:
  serena:
    security:
      mode: editing
      policies:
        - name: restrict-paths
          match:
            args:
              relative_path: { allow_prefix: ["src/", "internal/"] }
          action: deny
```

### Modes

Quick security profiles: `read-only` (blocks all write tools), `editing` (full access), `custom` (policies only).

```yaml
servers:
  ml-pipeline:
    security:
      mode: read-only   # prevents accidental changes to ML models
```

### Audit Logging

Every tool call is logged to a JSONL file with timestamps, arguments, and policy decisions:

```yaml
security:
  global:
    audit:
      enabled: true
      log: "$(mcpx.project_root)/.mcpx/audit.jsonl"
      redact: ["$(secret.*)"]
```

```json
{"timestamp":"2026-03-28T18:19:50Z","server":"serena","tool":"find_symbol","action":"allowed"}
{"timestamp":"2026-03-28T18:19:59Z","server":"serena","tool":"find_symbol","action":"denied","policy_name":"no-path-traversal"}
```

See [Security Documentation](https://codestz.github.io/mcpx/security/overview) for the full reference.

## Configuration

### Two-level config

| File | Scope | Commit? |
|------|-------|---------|
| `~/.mcpx/config.yml` | Global (user-level) | No |
| `.mcpx/config.yml` | Project-level | Yes |

Project config wins on conflict.

### Dynamic variables

```yaml
args:
  - "$(mcpx.project_root)"    # directory containing .mcpx/config.yml
  - "$(git.root)"              # git repository root
  - "$(env.API_KEY)"           # environment variable
  - "$(secret.github_token)"   # OS keychain secret
  - "$(sys.os)"                # linux, darwin, windows
```

### Server options

```yaml
servers:
  my-server:
    command: string          # executable to run (required)
    args: [string]           # command arguments
    transport: stdio|http|sse # protocol (default: stdio)
    daemon: bool             # keep server alive between calls
    startup_timeout: string  # e.g. "60s" (default: "30s")
    url: string              # for http/sse transports
    headers: {}              # HTTP headers
    auth:                    # authentication
      type: bearer
      token: "$(secret.token)"
    env: {}                  # extra environment variables
    security: {}             # per-server security config
```

## How AI Agents Use mcpx

Add this to your project's `CLAUDE.md` (or equivalent):

```markdown
## Tools

This project uses mcpx for MCP tool access:
- `mcpx list` — discover servers and tools
- `mcpx <server> <tool> --help` — get usage for any tool
- Call tools via Bash as needed
```

The AI discovers tools lazily, calls them on demand, and composes them through pipes.

## Secrets

```bash
mcpx secret set github_token ghp_abc123    # store in OS keychain
mcpx secret list                            # list secrets
mcpx secret remove github_token             # remove
```

Reference in config with `$(secret.<name>)` — resolved at runtime, never on disk, never logged.

## Daemon Mode

Heavy servers (like Serena with LSP) benefit from staying alive:

```yaml
servers:
  serena:
    daemon: true
```

```bash
mcpx daemon status             # show running daemons
mcpx daemon stop serena        # stop a daemon
mcpx ping serena               # health check with latency
```

## CLI Reference

```
mcpx <server> <tool> [flags]     Call a tool
mcpx <server> <tool> --help      Show tool help
mcpx <server> <tool> --stdin     Read args from stdin JSON
mcpx <server> --help             List all tools

mcpx <server> info               Server capabilities
mcpx <server> prompt list|<name> Prompts
mcpx <server> resource list|read Resources

mcpx list                        List servers
mcpx list <server> -v            List tools with flags
mcpx ping <server>               Health check
mcpx init                        Import .mcp.json

mcpx secret set|list|remove      Keychain secrets
mcpx daemon status|stop|stop-all Daemon management
mcpx version                     Print version
mcpx completion bash|zsh|fish    Shell completions
```

### Global flags

| Flag | Description |
|------|-------------|
| `--json` | Output raw JSON |
| `--quiet` | Suppress output |
| `--dry-run` | Show what would execute |
| `--pick <path>` | Extract a JSON field |
| `--timeout <dur>` | Per-call timeout |

### Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Tool error |
| 2 | Config error |
| 3 | Connection error |

## Architecture

```
cmd/mcpx/            entrypoint
internal/
  config/            YAML config loading, merging, validation
  security/          policy engine, audit logging
  resolver/          dynamic variable resolution: $(env.*), $(git.*), etc.
  mcp/               MCP protocol client (JSON-RPC 2.0 over stdio/http/sse)
  cli/               Cobra CLI, dynamic command generation from MCP schemas
  daemon/            daemon process management, unix socket transport
  secret/            OS keychain integration
```

- **Single binary**, zero runtime dependencies, sub-millisecond startup
- **Three transports**: stdio, HTTP (streamable), SSE
- **Security middleware**: policies evaluated before every tool call
- **Audit trail**: every call logged with secret redaction
- **No shell expansion** — `exec.Command`, not `sh -c`. No injection surface.
- **Secrets never on disk** — resolved from keychain at runtime
- **Daemon sockets** — unix socket at `/tmp/mcpx-<server>-<uid>.sock`, mode 0600

## Shell Completion

```bash
echo 'eval "$(mcpx completion bash)"' >> ~/.bashrc   # Bash
echo 'eval "$(mcpx completion zsh)"' >> ~/.zshrc     # Zsh
mcpx completion fish | source                         # Fish
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

[MIT](LICENSE)
