# Exit Codes

mcpx uses specific exit codes to indicate the type of failure.

## Codes

| Code | Name | Meaning |
|------|------|---------|
| `0` | Success | Command completed successfully |
| `1` | Tool Error | The MCP tool returned an error (`isError: true`) or Cobra command failed |
| `2` | Config Error | Configuration problem: bad YAML, missing server, invalid variable |
| `3` | Connection Error | Cannot reach the server: spawn failure, timeout, transport death |

## When Each Code Fires

### Exit 0 — Success

- Tool call completed and returned results
- `mcpx list` displayed servers
- `mcpx ping` got a response
- `mcpx secret set` stored the secret

### Exit 1 — Tool Error

- MCP tool returned `isError: true` in its response
- Unknown tool name: `mcpx serena nonexistent_tool`
- Flag parsing error: missing required flag
- General Cobra command errors

### Exit 2 — Config Error

- YAML parse error in config file
- Server name not found in config: `mcpx nonexistent --help`
- Variable resolution failed: `$(secret.missing_key)`
- Invalid variable namespace: `$(invalid.var)`

### Exit 3 — Connection Error

- Server command not found in PATH
- Server process exited during startup
- Startup timeout exceeded
- Daemon socket unreachable
- `mcpx ping` failure

## Usage in Scripts

```bash
mcpx ping serena --quiet
case $? in
  0) echo "Server is healthy" ;;
  3) echo "Server unreachable" ;;
  *) echo "Unexpected error" ;;
esac
```

```bash
# Fail fast if server is down
mcpx ping serena --quiet || exit 1
mcpx serena search_symbol --name "Auth"
```

## Usage by AI Agents

AI agents can use exit codes to decide next steps:

- **Exit 0**: parse stdout for results
- **Exit 1**: read stderr for tool error details, possibly retry with different arguments
- **Exit 2**: config problem — check `.mcpx/config.yml`
- **Exit 3**: server unreachable — try `mcpx daemon stop <server>` and retry, or report the issue
