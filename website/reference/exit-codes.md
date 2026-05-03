# Exit Codes

mcpx uses specific exit codes to indicate the type of failure.

## Codes

| Code | Name | Meaning |
|------|------|---------|
| `0` | Success | Command completed successfully |
| `1` | Tool Error | The MCP tool returned an error (`isError: true`), Cobra command failed, or `--validate-args` reported issues |
| `2` | Config Error | Configuration problem: bad YAML, missing server, invalid variable |
| `3` | Connection Error | Cannot reach the server: spawn failure, transport death, daemon socket unreachable |
| `4` | Timeout | `--timeout` exceeded for the call |
| `5` | Policy Denied | A security policy denied the call (see `mcpx <server>` audit log) |
| `6` | Tool Not Found | Tool name not in the server's `tools/list` |

Codes 4–6 were introduced in v1.6.0 to give agents and scripts finer-grained branching.

## When Each Code Fires

### Exit 0 — Success

- Tool call completed and returned results
- `mcpx list` displayed servers
- `mcpx ping` got a response
- `mcpx secret set` stored the secret
- `mcpx find` returned results
- `mcpx --validate-args` succeeded

### Exit 1 — Tool Error

- MCP tool returned `isError: true` in its response
- Flag parsing error: missing required flag, type mismatch
- `--validate-args` reported one or more issues
- General Cobra command errors

### Exit 2 — Config Error

- YAML parse error in config file
- Variable resolution failed: `$(secret.missing_key)`
- Invalid variable namespace: `$(invalid.var)`

### Exit 3 — Connection Error

- Server command not found in PATH
- Server process exited during startup
- Daemon socket unreachable
- `mcpx ping` failure
- HTTP/SSE transport unreachable

### Exit 4 — Timeout

- The call exceeded `--timeout`
- Surfaces as `context deadline exceeded` in stderr

### Exit 5 — Policy Denied

- A `deny` policy matched the tool name, arguments, or content
- The call is logged to the audit log if audit is enabled
- `mcpx doctor` can verify policy configuration

### Exit 6 — Tool Not Found

- Tool name doesn't appear in the server's `tools/list`
- mcpx prints a "did you mean…" suggestion when the typo is within Levenshtein distance 2

## Usage in Scripts

```bash
mcpx ping serena --quiet
case $? in
  0) echo "Server is healthy" ;;
  3) echo "Server unreachable" ;;
  4) echo "Timed out" ;;
  *) echo "Unexpected error" ;;
esac
```

```bash
# Fail fast if server is down
mcpx ping serena --quiet || exit 1
mcpx serena find_symbol --name_path_pattern "Auth"
```

## Usage by AI Agents

AI agents can use exit codes to decide next steps:

- **Exit 0**: parse stdout for results
- **Exit 1**: read stderr for tool error details; possibly retry with different arguments (consider `--example` to learn the correct shape)
- **Exit 2**: config problem — check `.mcpx/config.yml`, run `mcpx doctor`
- **Exit 3**: server unreachable — try `mcpx daemon stop <server>` and retry, or run `mcpx doctor`
- **Exit 4**: extend `--timeout` (e.g., `--timeout 120s`) and retry
- **Exit 5**: do not retry — a policy explicitly denied the call. The user must adjust policies if the call is legitimate.
- **Exit 6**: typo in the tool name — read the suggestion from stderr or run `mcpx find <intent>` to discover the right tool
