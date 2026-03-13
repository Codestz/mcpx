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

## Large Content: @file syntax

Any string flag accepts `@/path` to read from a file or `@-`/`-` to read from stdin:
```bash
mcpx <server> <tool> --body @/tmp/code.go   # Read file into --body
mcpx <server> <tool> --body @-              # Read stdin into --body
mcpx <server> <tool> --body -               # Same (backward compat)
```

## Output Extraction: --pick

Extract a JSON field from the result without jq:
```bash
mcpx <server> <tool> --pick field.path      # Dot-separated path
mcpx <server> <tool> --pick items.0.name    # Array index access
```

## Timeout Override: --timeout

Override the default call timeout for a single invocation:
```bash
mcpx <server> <tool> --timeout 60s          # Go duration format
```

## Stdin Merge

`--stdin` can be combined with CLI flags. Flags win on conflict:
```bash
echo '{"body":"content"}' | mcpx <server> <tool> --stdin --name_path Foo
```

## Tips for AI Agents

- Use `--body @/tmp/file` for large content to avoid shell escaping
- Use `--pick field` instead of piping through jq for single fields
- Combine `--stdin` with flags for mixed large+small arguments
- Use `--timeout 120s` for long-running operations
@SERENA.md
