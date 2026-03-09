# mcpx — MCP Server CLI Proxy

mcpx wraps MCP servers into CLI tools. Call them via Bash instead of loading schemas into context.

## Quick Reference

```bash
mcpx list                        # List configured servers
mcpx list <server> -v            # List all tools with flags
mcpx <server> --help             # Show server tools
mcpx <server> <tool> --help      # Show tool flags
mcpx <server> <tool> --flags     # Call a tool
mcpx <server> <tool> --stdin      # Read args from stdin JSON
mcpx <server> <tool> --json       # Output raw JSON
mcpx daemon status               # Show running daemons
```

## Configured Servers

- **serena** — `serena` (daemon)

## Usage Pattern

1. Discover: `mcpx <server> --help` to see available tools
2. Inspect: `mcpx <server> <tool> --help` to see flags
3. Call: `mcpx <server> <tool> --flag value`
4. For long args: `printf '{"key":"value"}' | mcpx <server> <tool> --stdin`
@SERENA.md
