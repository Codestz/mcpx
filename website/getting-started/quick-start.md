# Quick Start

Get from zero to calling MCP tools in under 2 minutes.

## Option A: Import Existing Config

If you already have MCP servers configured (`.mcp.json` from Claude Code, etc.):

```bash
mcpx init
```

```
Detected .mcp.json
Imported 3 servers:
  serena          uvx (daemon)
  seq-thinking    npx
  filesystem      npx
Created .mcpx/config.yml
```

Done. Skip to [Use It](#use-it).

## Option B: Manual Config

Create `.mcpx/config.yml` in your project root:

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

Or add a global server at `~/.mcpx/config.yml`:

```yaml
servers:
  sequential-thinking:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-sequential-thinking"]
    transport: stdio
```

## Use It

### Discover servers

```bash
mcpx list
#   serena              uvx (daemon)
#   sequential-thinking npx
```

### Discover tools

```bash
mcpx list serena -v
```

Shows every tool with all its flags — full discovery in one call.

### Call a tool

```bash
mcpx serena search_symbol --name "UserAuth"
```

### Get help for a tool

```bash
mcpx serena search_symbol --help
```

Shows flag names, types, required markers, descriptions, and defaults — all auto-generated from the MCP schema.

### Inspect server capabilities

```bash
mcpx serena info
# Shows server name, version, protocol, and supported capabilities
```

### Explore prompts and resources

If the server supports them:

```bash
mcpx <server> prompt list                       # list prompts
mcpx <server> prompt <name> --arg value         # get a prompt
mcpx <server> resource list                     # list resources
mcpx <server> resource read <uri>               # read a resource
```

### Health check

```bash
mcpx ping serena
# serena: ok (21 tools, 47ms)
```

### JSON output

```bash
mcpx serena search_symbol --name "Auth" --json
```

### Pipe and compose

```bash
mcpx serena search_symbol --name "Auth" | jq -r '.[].file' | xargs code
```

## What Just Happened

1. `mcpx` read your config to find the server
2. For daemon servers: spawned the server once, connected via unix socket
3. For direct servers: spawned a subprocess, did the MCP handshake
4. Translated your CLI flags into a JSON-RPC `tools/call` request
5. Printed the result to stdout

The AI agent does exactly the same thing — it just runs these commands via Bash.

## Next Steps

- [AI Agent Setup](/getting-started/ai-agent-setup) — integrate with Claude Code or Cursor
- [Configuration](/guide/configuration) — two-level config, server options
- [Daemon Mode](/guide/daemon-mode) — keep heavy servers warm
