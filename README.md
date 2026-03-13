```
                         __  __  ____ ____  __  __
                        |  \/  |/ ___|  _ \ \ \/ /
                        | |\/| | |   | |_) | \  /
                        | |  | | |___|  __/  /  \
                        |_|  |_|\____|_|    /_/\_\

             MCP servers as CLI tools — built for AI agents.
```

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/codestz/mcpx)](https://goreportcard.com/report/github.com/codestz/mcpx)

**[Documentation](https://codestz.github.io/mcpx)** | **[Installation](https://codestz.github.io/mcpx/getting-started/installation)** | **[GitHub](https://github.com/codestz/mcpx)**

---

**mcpx** wraps any [MCP](https://modelcontextprotocol.io) server into a CLI that AI agents call through Bash. Instead of loading tool schemas into context (50-100K tokens per session), the AI runs shell commands on demand.

Zero context overhead. On-demand discovery. Full UNIX composability.

```bash
# Instead of loading 5 MCP servers into context (~100K tokens)...
# The AI just runs commands when it needs them:

mcpx serena search_symbol --name "UserAuth"
mcpx sequential-thinking think --problem "how to scale this service"
mcpx filesystem read_file --path ./src/main.go | grep "TODO"
```

## The Problem

When you configure MCP servers for an AI agent (Claude Code, Cursor, etc.), every server's tool definitions get loaded into the conversation context **upfront**:

| Setup | Context Cost | What Happens |
|-------|-------------|--------------|
| 3 MCP servers | ~30-60K tokens | Slow init, bloated context |
| 10 MCP servers | ~100K+ tokens | Context nearly full before any work |
| **mcpx (any count)** | **0 tokens** | **Tools called on demand via Bash** |

The AI carries those definitions for the entire session — whether it uses them or not.

## The Solution

Give the AI one tool it already knows: **the terminal**.

```bash
# Discover what's available (only when needed)
mcpx list                              # list servers
mcpx list serena -v                    # list tools with all flags

# Call tools on demand
mcpx serena search_symbol --name "Auth"

# Get help for a specific tool
mcpx serena search_symbol --help

# Compose tools — the AI already knows how to pipe
mcpx serena search_symbol --name "Auth" | jq -r '.[].file'
```

### Before mcpx

```
Session starts → Load 5 servers → 80K tokens gone → AI works with reduced context
```

### After mcpx

```
Session starts → 0 tokens → AI calls mcpx via Bash when needed → Full context available
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
cd mcpx
make build
```

## Quick Start

### 1. Import existing MCP config

If you already have MCP servers configured (`.mcp.json`, Claude Code, etc.):

```bash
mcpx init
# Detected .mcp.json — imported 3 servers: serena, sequential-thinking, filesystem
# Created .mcpx/config.yml
```

### 2. Or configure manually

**Project config** (`.mcpx/config.yml` — commit this):

```yaml
servers:
  serena:
    command: uvx
    args:
      - --from
      - git+https://github.com/oraios/serena
      - serena
      - start-mcp-server
      - --project
      - "$(mcpx.project_root)"
    transport: stdio
    daemon: true
```

**Global config** (`~/.mcpx/config.yml` — personal, not committed):

```yaml
servers:
  sequential-thinking:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-sequential-thinking"]
    transport: stdio
```

### 3. Use it

```bash
mcpx serena search_symbol --name "UserAuth"
mcpx ping serena                           # health check
mcpx serena search_symbol --name "Auth" | jq -r '.[].file' | xargs code
```

## How AI Agents Use mcpx

Add this to your project's `CLAUDE.md` (or equivalent agent instructions):

```markdown
## Tools

This project uses mcpx for MCP tool access:
- `mcpx list` — discover servers and tools
- `mcpx <server> <tool> --help` — get usage for any tool
- Call tools via Bash as needed
```

The AI discovers tools lazily, calls them on demand, and composes them through pipes. Context stays clean.

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
```

| Variable | Resolves to |
|----------|------------|
| `$(mcpx.project_root)` | Directory containing `.mcpx/config.yml` |
| `$(mcpx.cwd)` | Current working directory |
| `$(mcpx.home)` | User home directory |
| `$(git.root)` | Git repository root |
| `$(git.branch)` | Current branch |
| `$(git.remote)` | Remote URL |
| `$(git.commit)` | Current commit hash |
| `$(env.*)` | Any environment variable |
| `$(secret.*)` | OS keychain secret (see [Secrets](#secrets)) |
| `$(sys.os)` | `linux`, `darwin`, `windows` |
| `$(sys.arch)` | `amd64`, `arm64` |

### Server options

```yaml
servers:
  my-server:
    command: string          # executable to run (required)
    args: [string]           # command arguments
    transport: stdio         # stdio (default)
    daemon: bool             # keep server alive between calls (default: false)
    startup_timeout: string  # e.g. "60s" (default: "30s")
    env:                     # extra environment variables
      KEY: "$(secret.key)"
```

## Secrets

Manage secrets stored in the OS keychain:

```bash
mcpx secret set github_token ghp_abc123    # store a secret
mcpx secret list                            # list stored secrets
mcpx secret remove github_token             # remove a secret
```

Reference secrets in config with `$(secret.<name>)` — they're resolved at runtime, never written to disk, never logged.

## Daemon Mode

Heavy MCP servers (like Serena with LSP) benefit from staying alive between calls:

```yaml
servers:
  serena:
    command: uvx
    args: [...]
    daemon: true    # keeps server warm
```

```bash
mcpx daemon status             # show running daemons
mcpx daemon stop serena        # stop a daemon
mcpx daemon stop-all           # stop all daemons
mcpx ping serena               # health check with latency
```

When `daemon: true`, mcpx spawns the server once and connects via unix socket on subsequent calls. Zero spawn cost per invocation.

## CLI Reference

```
mcpx <server> <tool> [flags]     Call a tool
mcpx <server> <tool> --help      Show tool help with all flags
mcpx <server> <tool> --stdin     Read arguments from stdin as JSON
mcpx <server> --help             List all tools for a server

mcpx list                        List configured servers
mcpx list <server> -v            List tools with all flags
mcpx ping <server>               Health check a server
mcpx init                        Import .mcp.json into .mcpx/config.yml
mcpx configure                   Generate tool docs for CLAUDE.md

mcpx secret set <n> <v>          Store a secret in OS keychain
mcpx secret list                 List stored secrets
mcpx secret remove <name>        Remove a secret

mcpx daemon status               Show running daemons
mcpx daemon stop <server>        Stop a daemon
mcpx daemon stop-all             Stop all daemons

mcpx version                     Print version
mcpx completion bash|zsh|fish    Generate shell completions
```

### v1.1.0 flags

| Flag | Description |
|------|-------------|
| `--pick <path>` | Extract a JSON field from output (e.g. `--pick 0.name`) |
| `--timeout <dur>` | Per-call timeout override (e.g. `--timeout 60s`) |
| `@/path` | Any string flag reads value from file (e.g. `--body @/tmp/code.go`) |

`--stdin` now merges with CLI flags (flags win on conflict).

### Global flags

| Flag | Description |
|------|-------------|
| `--json` | Output raw JSON |
| `--quiet` | Suppress output |
| `--dry-run` | Show what would execute |

### Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Tool error |
| 2 | Config error |
| 3 | Connection error |

## Shell Completion

```bash
# Bash
echo 'eval "$(mcpx completion bash)"' >> ~/.bashrc

# Zsh
echo 'eval "$(mcpx completion zsh)"' >> ~/.zshrc

# Fish
mcpx completion fish | source
```

Completions include dynamic server and tool names from your config.

## Architecture

```
cmd/mcpx/            entrypoint
internal/
  config/            YAML config loading, merging, validation
  resolver/          dynamic variable resolution: $(env.*), $(git.*), etc.
  mcp/               MCP protocol client (JSON-RPC 2.0 over stdio)
  cli/               Cobra CLI, dynamic command generation from MCP schemas
  daemon/            daemon process management, unix socket transport
  secret/            OS keychain integration
```

- **Single binary**, zero runtime dependencies, sub-millisecond startup
- **No shell expansion** — `exec.Command`, not `sh -c`. No injection surface.
- **Secrets never on disk** — resolved from keychain at runtime, injected into process env
- **Daemon sockets** — unix socket at `/tmp/mcpx-<server>-<uid>.sock`, mode 0600

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

[MIT](LICENSE)
