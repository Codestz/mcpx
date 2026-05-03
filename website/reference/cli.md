# CLI Reference

Complete command reference for mcpx.

## Tool Calling

### `mcpx <server> <tool> [flags]`

Call an MCP tool.

```bash
mcpx serena search_symbol --name "Auth"
mcpx serena find_file --file_mask "*.go" --relative_path "."
```

Flags are auto-generated from the tool's MCP schema. Use `--help` to see them.

### `mcpx <server> <tool> --help`

Show help for a specific tool, including all flags with types, required status, descriptions, and defaults.

```bash
mcpx serena search_symbol --help
```

### `mcpx <server> <tool> --stdin`

Read tool arguments from stdin as a JSON object. CLI flags are merged on top (flags win on conflict).

```bash
echo '{"name": "Auth", "include_body": true}' | mcpx serena find_symbol --stdin

# Merge: stdin provides body, flag overrides name_path
echo '{"body":"..."}' | mcpx serena replace_symbol_body --stdin --name_path Foo
```

### `@file` syntax

Any string flag accepts `@/path` to read its value from a file, or `@-`/`-` to read from stdin:

```bash
mcpx serena replace_symbol_body --body @/tmp/handler.go --name_path HandleAuth
cat handler.go | mcpx serena replace_symbol_body --body @- --name_path HandleAuth
```

### `--pick <path>`

Extract a value from JSON output without jq. Supports dot-separated paths and array indices.

```bash
mcpx serena find_symbol --name_path_pattern "User*" --pick 0.name
# → UserAuth
```

### `--timeout <duration>`

Override the default call timeout for a single invocation. Uses Go duration format.

```bash
mcpx serena search_for_pattern --substring_pattern "TODO" --timeout 60s
```

### `mcpx <server> <tool> --example`

Print a JSON skeleton for the tool's arguments based on the normalized schema. Required fields get placeholder values; optional fields use schema defaults or enum head.

```bash
mcpx serena find_symbol --example
# {
#   "name_path_pattern": "<string>",
#   "depth": 0,
#   "include_body": false,
#   ...
# }
```

Pair with `--json` for piping into other tools.

### `mcpx <server> <tool> --validate-args`

Type-check arguments without invoking the tool. Reports missing required fields, wrong types, and enum mismatches. Exits `0` on success, `1` on issues.

```bash
mcpx serena find_symbol --name_path_pattern Auth --validate-args
# ok
```

### `mcpx <server> --help`

List tools for a server. Default is one line per tool with required flags surfaced — designed to fit a server with 20+ tools in ~30 lines.

```bash
mcpx serena --help            # compact (default)
mcpx serena --help --full     # verbose with descriptions
```

---

## Server Info

### `mcpx <server> info`

Show server metadata and capabilities: server name, version, protocol version, and a checklist of supported MCP primitives.

```bash
mcpx serena info
# serena
#
# Server:   FastMCP 1.23.0
# Protocol: 2025-06-18
#
# Capabilities:
#   [x] Tools
#   [x] Prompts
#   [x] Resources
#   [ ] Logging

mcpx serena info --json
```

---

## Prompts

MCP servers can expose prompt templates that generate structured messages.

### `mcpx <server> prompt list`

List all available prompts.

```bash
mcpx everything prompt list
```

### `mcpx <server> prompt <name> --help`

Show a prompt's arguments with required markers.

```bash
mcpx everything prompt args-prompt --help
```

### `mcpx <server> prompt <name> [--arg value ...]`

Get a prompt with arguments. Required arguments are validated before the request is sent.

```bash
mcpx everything prompt simple-prompt
# [user]: This is a simple prompt without arguments.

mcpx everything prompt args-prompt --city "Tokyo"
# [user]: What's weather in Tokyo?
```

---

## Resources

MCP servers can expose resources — static content or dynamic URI templates.

### `mcpx <server> resource list`

List all resources and resource templates.

```bash
mcpx everything resource list
```

### `mcpx <server> resource read <uri>`

Read a resource by URI. Text resources are printed directly; binary resources show `[binary: mime, N bytes]`.

```bash
mcpx everything resource read "demo://resource/static/document/architecture.md"
```

---

## Discovery

### `mcpx list`

List all configured servers.

```bash
mcpx list
#   serena              uvx (daemon)
#   sequential-thinking npx
```

### `mcpx list <server>`

List tools for a specific server.

```bash
mcpx list serena
```

### `mcpx list <server> -v`

List tools with all their flags — full discovery in one call.

```bash
mcpx list serena -v
```

### `mcpx ping <server>`

Health check a server. Connects, lists tools, reports latency.

```bash
mcpx ping serena
# serena: ok (21 tools, 47ms)

mcpx ping serena --json
# {"server":"serena","status":"ok","tools":21,"ms":47}
```

Exit code 3 on failure.

### `mcpx find <query>`

BM25-ranked tool search across every configured server. Use this when you don't know which tool you need — returns the top candidates in ~80 tokens instead of 5–15K from `list -v`.

```bash
mcpx find "search code by regex"
# query  "search code by regex"
# found  4 result(s) across 1 server(s)
#
#   serena.search_for_pattern   1.00  Offers a flexible search for arbitrary patterns...
#   serena.find_symbol          0.34  Retrieves info on symbols/code entities...
#   ...

mcpx find "github issue" --top 3 --json
mcpx find "..." --server serena   # restrict to one server
```

Indexed from the schema cache; first run per server populates the cache.

---

## Batching

### `mcpx batch`

Run many tool calls in parallel from NDJSON on stdin. One mcpx process, one MCP client per server reused across the entire batch — no handshake-per-call overhead.

```bash
printf '%s\n%s\n' \
  '{"id":"a","server":"serena","tool":"find_symbol","args":{"name_path_pattern":"Auth"}}' \
  '{"id":"b","server":"serena","tool":"find_symbol","args":{"name_path_pattern":"Token"}}' | mcpx batch
```

Output is NDJSON in input order with `id`, `ok`, `latency_ms`, `cached`, `result` or `error`.

Flags:
- `--parallel` (default) / `--sequential`
- `--max-concurrent N` (default = NumCPU)
- `--stop-on-error` — abort on first failure
- `--cache=false` — bypass result cache

The result cache deduplicates identical calls within the batch when enabled.

---

## Observability

### `mcpx gain`

Premium terminal dashboard for token-savings analytics. Reads `~/.mcpx/stats.jsonl` (every call writes one line transparently).

```bash
mcpx gain                        # current project, last 7 days
mcpx gain --all                  # every project
mcpx gain --since 24h            # narrow window
mcpx gain --by tool              # ranked tool table
mcpx gain --by server            # ranked server table
mcpx gain --by day               # daily breakdown
mcpx gain --history 20           # last N calls
mcpx gain --suggest              # mined recommendations
mcpx gain --watch                # live refresh
mcpx gain --json                 # machine-readable
```

The dashboard shows the hero "tokens saved" metric, sparkline, top tools by calls, top tools by savings, and the most recent calls with their latency + cache state.

### `mcpx ui`

Always-on web dashboard daemon. Auto-spawned on the first tool call (token-protected, 127.0.0.1 only, idle-shutdown after 1h). The URL is printed once per shell session.

```bash
mcpx ui status                   # show URL or "inactive"
mcpx ui open                     # print URL (use with `open $(mcpx ui open)`)
mcpx ui stop                     # stop the daemon
mcpx ui disable                  # how to disable permanently
```

Opt out via `MCPX_UI=off` env var or `ui.enabled: false` in config.

---

## Diagnostics

### `mcpx doctor`

Run a series of checks against config and connectivity: YAML validity, command paths, secret resolution, daemon liveness, MCP `initialize` handshake, `tools/list` reachability.

```bash
mcpx doctor
#   mcpx doctor
#   ─────────────────────────────────────
#     [ok]   config         loaded 1 server(s), project = /path
#
#   serena
#     [ok]   command        /path/to/serena
#     [ok]   daemon         running, socket reachable
#     [ok]   initialize     FastMCP 1.23.0
#     [ok]   tools/list     21 tool(s)
#
#   all checks passed

mcpx doctor --json               # machine-readable
```

Exits `0` on all-ok, `2` on any failure.

---

## Configuration

### `mcpx init`

Import servers from `.mcp.json` (Claude Code format) into `.mcpx/config.yml`.

```bash
mcpx init
```

### `mcpx configure`

Generate the agent-facing reference docs that teach Claude Code (and any mcpx-aware coding agent) how to use the configured MCP servers. Idempotent — re-run after adding/removing servers.

Writes:
- `.claude/MCPX.md` — composition primitives, discovery, exit codes
- `.claude/<SERVER>.md` — per-server tool selector table + compact reference
- `.claude/CLAUDE.md` — adds `@MCPX.md` and `@<SERVER>.md` references so the agent loads them automatically

```bash
mcpx configure                   # project (.claude/)
mcpx configure --global          # user-level (~/.claude/)
```

The format is now agent-optimized by default: tabular, decision-oriented, no human-ops content (caching internals, observability, dashboard live in the CLI commands above when a human needs them).

---

## Secrets

### `mcpx secret set <name> <value>`

Store a secret in the OS keychain.

```bash
mcpx secret set github_token ghp_abc123
```

### `mcpx secret list`

List stored secret names.

```bash
mcpx secret list
```

### `mcpx secret remove <name>`

Remove a secret. Aliases: `rm`, `delete`.

```bash
mcpx secret remove github_token
```

---

## Daemon Management

### `mcpx daemon status`

Show running daemons.

```bash
mcpx daemon status
#   serena  running  /tmp/mcpx-serena-501.sock
```

### `mcpx daemon stop <server>`

Stop a specific daemon.

```bash
mcpx daemon stop serena
```

### `mcpx daemon stop-all`

Stop all running daemons.

```bash
mcpx daemon stop-all
```

---

## Utility

### `mcpx version`

Print the mcpx version.

### `mcpx completion <shell>`

Generate shell completion script. Supported: `bash`, `zsh`, `fish`, `powershell`.

---

## Global Flags

These flags work with any command:

| Flag | Description |
|------|-------------|
| `--json` | Output raw JSON |
| `--quiet` | Suppress all output |
| `--dry-run` | Show what would execute without running |
| `--pick <path>` | Extract a JSON field from output (v1.1) |
| `--timeout <dur>` | Per-call timeout override (v1.1) |
