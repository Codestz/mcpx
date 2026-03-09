# Configuration

mcpx uses a two-level YAML configuration system.

## Config Files

| File | Scope | Commit to git? |
|------|-------|----------------|
| `.mcpx/config.yml` | Project-level | Yes |
| `~/.mcpx/config.yml` | Global (user-level) | No |

mcpx searches for project config by walking up from the current directory until it finds a `.mcpx/` directory.

## Merge Behavior

When both configs exist, **project config wins at the server level**. If both define a server named `serena`, the project version is used entirely — fields are not merged within a server.

```yaml
# ~/.mcpx/config.yml (global)
servers:
  serena:
    command: uvx
    args: ["--from", "serena@1.0", "serena"]
  my-tool:
    command: my-tool-binary

# .mcpx/config.yml (project)
servers:
  serena:
    command: uvx
    args: ["--from", "serena@2.0", "serena", "--project", "$(mcpx.project_root)"]
    daemon: true
```

Result: `serena` uses the project version (v2.0 with daemon). `my-tool` comes from global.

## Server Options

```yaml
servers:
  my-server:
    # Required
    command: string              # executable to run

    # Optional
    args: [string]               # command arguments (supports variables)
    transport: stdio             # "stdio" (default)
    daemon: bool                 # keep alive between calls (default: false)
    startup_timeout: string      # e.g. "60s" (default: "30s")
    env:                         # extra environment variables
      KEY: value                 # supports $(variable) syntax
```

### `command`

The executable to run. Must be in PATH or an absolute path.

```yaml
command: uvx                     # in PATH
command: /usr/local/bin/my-tool  # absolute path
command: npx                     # common for Node.js MCP servers
```

### `args`

Arguments passed to the command. Supports [dynamic variables](/guide/variables).

```yaml
args:
  - --from
  - git+https://github.com/oraios/serena
  - serena
  - start-mcp-server
  - --project
  - "$(mcpx.project_root)"
```

### `transport`

Communication protocol. Currently supported: `stdio`.

### `daemon`

When `true`, the server is spawned once and kept alive. Subsequent calls connect via unix socket instead of spawning a new process. See [Daemon Mode](/guide/daemon-mode).

### `startup_timeout`

How long to wait for the server to become responsive. Useful for servers that need time to initialize (e.g., Serena loading LSP indexes).

```yaml
startup_timeout: "60s"           # default: "30s"
```

### `env`

Extra environment variables passed to the server process. Supports [dynamic variables](/guide/variables).

```yaml
env:
  GITHUB_TOKEN: "$(secret.github_token)"
  NODE_ENV: production
```

## Import from .mcp.json

If you have an existing `.mcp.json` (Claude Code format):

```bash
mcpx init
```

This parses the file and generates `.mcpx/config.yml` automatically.

## Interactive Configuration

```bash
mcpx configure
```

Guides you through adding a new server interactively.

## Validation

mcpx validates config on load:

- `command` is required for every server
- Variable references must use valid namespaces
- YAML must be well-formed

Invalid config produces exit code 2 with a descriptive error message.
