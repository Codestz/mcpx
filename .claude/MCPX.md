# mcpx

Call MCP tools through `mcpx <server> <tool> --flags`. Don't load native MCP — use the CLI commands below.

## Servers

- **serena** *(daemon)*

## Compose

| Need | How |
|---|---|
| Standard call | `mcpx <server> <tool> --flag value` |
| Large arg from file | `--body @/path/to/file` |
| Read body from stdin | `--body @-` or `--body -` |
| Pass full args as JSON | `printf '{...}' \| mcpx <server> <tool> --stdin` |
| Mix stdin + flags | `--stdin --flag value` (flags win) |
| Extract one JSON field | `--pick path.to.field` |
| Raw JSON output | `--json` |
| Per-call timeout | `--timeout 60s` (Go duration) |
| Show resolved command | `--dry-run` |
| Args skeleton | `mcpx <server> <tool> --example` |
| Type-check args | `mcpx <server> <tool> --validate-args ...` |

## Discover

| Need | How |
|---|---|
| Find the right tool by intent | `mcpx find "<query>"` |
| One-line list of a server's tools | `mcpx <server> --help` |
| Full schema for one tool | `mcpx <server> <tool> --help` |
| Run many tool calls in parallel | `mcpx batch < calls.jsonl` |

## Exit codes

`0` ok · `1` tool error · `2` config · `3` connection · `4` timeout · `5` policy denied · `6` tool not found.
@SERENA.md
