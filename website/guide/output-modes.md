# Output Modes

mcpx supports three output modes via global flags.

## Pretty (default)

Human-readable text with color. Used when no flags are specified.

```bash
mcpx serena search_symbol --name "Auth"
# Symbol: AuthService
# File: src/auth/service.go
# Kind: Class
```

```bash
mcpx ping serena
# serena: ok (21 tools, 47ms)
```

Colors are automatically disabled when output is piped or when `NO_COLOR` is set.

## JSON (`--json`)

Raw JSON output. Useful for AI agents parsing results or piping to `jq`.

```bash
mcpx serena search_symbol --name "Auth" --json
```

```json
{
  "content": [
    {
      "type": "text",
      "text": "Symbol: AuthService..."
    }
  ],
  "isError": false
}
```

```bash
mcpx ping serena --json
```

```json
{
  "server": "serena",
  "status": "ok",
  "tools": 21,
  "ms": 47
}
```

```bash
mcpx list --json
```

```json
{
  "serena": {
    "command": "uvx",
    "transport": "stdio",
    "daemon": true
  }
}
```

## Quiet (`--quiet`)

Suppresses all output. Useful in scripts where you only care about the exit code.

```bash
mcpx ping serena --quiet
echo $?
# 0
```

## Dry Run (`--dry-run`)

Shows what would execute without actually running it. Useful for debugging config and variable resolution.

```bash
mcpx serena search_symbol --name "Auth" --dry-run
```

```
Dry run — nothing will be executed

Server:  serena
Tool:    search_symbol
Command: uvx --from git+https://github.com/oraios/serena serena start-mcp-server --project /home/user/myproject
Arguments:
  {"name": "Auth"}
```

## Exit Codes

| Code | Meaning | When |
|------|---------|------|
| 0 | Success | Tool call completed |
| 1 | Tool error | MCP tool returned an error |
| 2 | Config error | Bad YAML, missing server, bad variable |
| 3 | Connection error | Server won't start, timeout, transport failure |

See [Exit Codes](/reference/exit-codes) for details.
