# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.6.0] - 2026-05-03

The "Agentic Supremacy" release. mcpx becomes a measurable, observable, and intelligent control plane: every call is recorded, every schema is cached, every server is one `mcpx find` away, and an always-on dashboard makes ROI visible.

### Added — Foundation

- **JSONL stats** (`internal/stats/`) — every tool call writes one line to `~/.mcpx/stats.jsonl`: timestamp, args, latency, cache hits, exit code, tokens saved vs native MCP loading. Async writer; never blocks the caller.
- **Schema cache** (`internal/schemacache/`) — `tools/list` (and prompts/resources) cached at `~/.mcpx/cache/schemas/<key>.json` with TTL. First call per server is slow; everything after is instant. Computes `native_baseline_tokens` once at populate time so the stats writer can quote real savings.
- **Result cache** — idempotent calls (`get_*`/`list_*`/`find_*`/`search_*`/`read_*` heuristic, plus explicit allow-list) deduplicate within `cache.result_ttl` (default 30s). **File mtime participates in the cache key** when args contain a `relative_path` (or `path`/`file`/`file_path`) — editing the underlying file shifts the key and forces a fresh fetch, so cached reads cannot return stale data after an edit.
- **Schema normalizer** — handles JSON Schema union types (`type: ["string","null"]`), `oneOf`/`anyOf`/`allOf`, `$ref`; unknown keywords preserved in `Ext`. Fixes #14 (Sentry MCP server initialization).

### Added — Agent superpowers

- **`mcpx find <query>`** — BM25-ranked tool search across every configured server. ~80 tokens for top candidates instead of 5–15K tokens for `mcpx list -v`.
- **`mcpx batch`** — NDJSON in/out, parallel by default, single client per server reused across the entire batch.
- **`mcpx <server> <tool> --example`** — JSON skeleton with placeholder values from the normalized schema.
- **`mcpx <server> <tool> --validate-args ...`** — type/required check without invoking the tool.
- **Default-compact help** — `mcpx <server> --help` is one line per tool with required flags surfaced. Add `--full` for descriptions.
- **Typo remediation** — Levenshtein-2 "did you mean…" suggestions on tool-not-found and unknown-flag errors.
- **Pre/post edit guards** — `replace_symbol_body` calls are warned when the body starts with a duplicate declaration keyword (Go, Python, TS, JS, Rust, Ruby, Java, Kotlin, Swift). For `.go` files the modified file is re-parsed and `file:line:col` is surfaced when the edit broke syntax.

### Added — Operator surfaces

- **`mcpx gain`** — premium terminal dashboard: hero metric (tokens saved), 7-day sparkline, top-tools bars, top-savers, server health, recent calls. Subcommands: `--by tool|server|day`, `--history N`, `--suggest`, `--watch`, `--all`, `--project`, `--since`, `--json`.
- **Always-on web dashboard** — auto-spawned on first call. Token-protected, 127.0.0.1-only, idle-shutdown after 1h, opt-out via `MCPX_UI=off` or `ui.enabled: false`. Single-page UI: project sidebar, time range tabs (1h/24h/7d/30d/all), token efficiency bar, top tools, server health, click-to-inspect drawer, SSE live tail, regex filter (`/`).
- **`mcpx ui status|stop|open|disable`** — dashboard daemon controls.
- **`mcpx doctor`** — config + command path + secret resolution + daemon liveness + initialize + tools/list checks. `--json` for machine-readable output.

### Changed

- **Structured exit codes** — `0` ok · `1` tool error · `2` config · `3` connection · `4` timeout · `5` policy denied · `6` tool not found.
- **`mcpx version`** warns loudly on `+dirty` builds (so leaked debug code from local builds doesn't silently propagate).
- **`mcpx configure`** rewritten — generates an agent-grounded `MCPX.md` (composition + discover + exit codes) and per-server `<SERVER>.md` files (tool-selector table + compact reference). The `--format compact` flag is now hidden; output is agent-optimized by default. Edit-tool safety guidance only emitted for servers that expose body-mutation tools.
- Client version bumped to `1.6.0` in the MCP handshake.

### New packages

- `internal/stats/` — JSONL writer (async, drop-on-overflow), reader, aggregator (top-K, p95, daily buckets, hit rates).
- `internal/schemacache/` — schema cache, result cache, idempotence detection.
- `internal/find/` — BM25 ranker (snake/camel splitting, name-coverage bonus, plural stem).
- `internal/render/` — terminal primitives (Box, Bar, Sparkline, FormatNumber/Duration/Percent, term width).
- `internal/ui/` — dashboard daemon (lazy supervisor, HTTP, SSE, embedded HTML/CSS/JS).

### Config additions

```yaml
gain:
  enabled: true
  tokenizer: estimate     # bytes ÷ 4; honest approximation of Claude tokenization
  retain_days: 30
  stats_path: ""          # default: ~/.mcpx/stats.jsonl

cache:
  schema_ttl: 5m
  result_ttl: 30s
  result_enabled: true
  result_heuristic: false
  result_idempotent_tools: []   # e.g. ["serena.find_symbol"]

ui:
  enabled: true
  port: 7878              # 0 = ephemeral
  bind: 127.0.0.1
  idle_timeout: 1h
```

Environment overrides: `MCPX_AGENT`, `MCPX_CACHE`, `MCPX_UI`, `MCPX_VERBOSE`.

## [1.5.0] - 2026-03-30

### Added

- **Scoped daemons** — each project gets its own daemon process, preventing cross-session races. Two Claude sessions on different projects no longer share a Serena daemon. Scope is a hash of the project root path, embedded in socket/PID/log filenames.
- **Daemon discovery** — `daemon status` now discovers all running scoped daemons via PID file globbing. `daemon stop` and `daemon stop-all` handle scoped daemons correctly.
- **Security policy engine** — evaluate tool calls against configurable policies before they reach the server. Supports tool name globs, argument inspection (deny/allow prefix, deny pattern), and content regex matching (e.g. SQL mutation blocking). Global policies cascade with per-server overrides.
- **Security modes** — one-line presets: `read-only` (blocks all write tools), `editing` (full access), `custom` (policies only). Per-server.
- **Audit logging** — every tool call recorded in JSONL with timestamps, server name, tool, arguments, and policy decisions. Configurable log path with variable resolution. Secret redaction for sensitive values.
- **Allowed/blocked tool lists** — per-server whitelist (`allowed_tools`) and blacklist (`blocked_tools`) with glob pattern support.
- **Content matching** — `content.deny_pattern` and `content.require_pattern` with conditional `when` clause for inspecting argument values (SQL queries, code snippets, file paths).

### Changed

- **Branding** — repositioned from "MCP servers as CLI tools" to "Secure gateway for MCP servers — from CLI to production".
- **Website redesigned** — new landing page with full-width layout, security showcase, animated terminal demos, 4-column feature grid. New sections: Security (5 pages), Integrations (5 pages).
- **Config schema expanded** — `security` (top-level and per-server) fields added to `ServerConfig`.
- **README rewritten** — three-pillar structure (context cost, security, multi-server management) with YAML examples and live denial output.
- `SocketPath`, `PIDPath`, `LogPath` accept a `scope` parameter for project isolation.
- Client version bumped to `1.5.0` in MCP handshake.

### New packages

- `internal/security/` — policy engine (`Evaluator`), audit logger (`AuditLogger`), action types, result types.

## [1.3.0] - 2026-03-14

### Added

- **Prompts support** — `mcpx <server> prompt list`, `mcpx <server> prompt <name> --help`, `mcpx <server> prompt <name> [--arg value ...]`. Full MCP `prompts/list` and `prompts/get` with pagination.
- **Resources support** — `mcpx <server> resource list`, `mcpx <server> resource read <uri>`. Full MCP `resources/list`, `resources/templates/list`, and `resources/read` with pagination.
- **Server info** — `mcpx <server> info` shows server name, version, protocol version, and a capability checklist (tools, prompts, resources, logging).
- **Daemon capability forwarding** — daemon now caches the `InitializeResult` from startup and replays it to connecting clients. `info`, `--help`, and capability-based features now work correctly in daemon mode.
- **Resource templates** — `resource list` shows both static resources and URI templates.
- **Prompt argument validation** — required prompt arguments are validated client-side before the RPC call, with actionable error messages.
- **Enhanced server help** — `mcpx <server> --help` now shows available prompts and resources alongside tools, with usage hints for all subcommands.
- **Generate v2** — `mcpx configure` and `mcpx <server> generate` now include prompts and resources sections in generated documentation.
- **Shell completions** — `info`, `prompt`, `resource` added to tab-completion suggestions.

### Changed

- Server capabilities struct extended with `Prompts` and `Resources` fields.
- Client version bumped to `1.3.0` in MCP handshake.

## [1.2.0] - 2026-03-13

### Added

- **Streamable HTTP transport** — `transport: http` with POST-based JSON-RPC, dual content-type handling (application/json + text/event-stream), session management via `Mcp-Session-Id`, and DELETE on close.
- **Legacy SSE transport** — `transport: sse` with persistent GET stream for responses, POST for requests, and endpoint discovery via SSE events.
- **Protocol version upgrade** — updated from `2024-11-05` to `2025-11-25` (MCP spec compliance).
- **Server capabilities parsing** — `ServerCapabilities` and `ServerInfo` parsed from Initialize response.
- **Paginated tools/list** — cursor-based pagination support for servers with many tools.
- **Ping** — `client.Ping()` method for server health checks.
- **Expanded content types** — image, audio, and embedded resource content blocks with base64 display.
- **Headers and auth** — `headers` map and `auth.type: bearer` / `auth.token` config fields for remote transports.
- **URL variable resolution** — `$(var)` patterns resolved in server URLs and headers.

### Changed

- Go minimum bumped to 1.26.1 to fix stdlib vulnerabilities.

## [1.1.0] - 2026-03-13

### Added

- **`@file` syntax** — any string flag accepts `@/path` to read value from a file, or `@-`/`-` to read from stdin. Eliminates shell escaping for multi-line content.
- **`--pick`** — lightweight JSON field extraction without jq. Supports dot-separated paths and array indexing (e.g. `--pick items.0.name`).
- **`--timeout`** — per-call timeout override in Go duration format (e.g. `--timeout 60s`). Prevents hanging on slow MCP servers.
- **stdin merge** — `--stdin` now merges with CLI flags instead of replacing them. CLI flags win on conflict, so you can pipe large JSON while overriding specific fields.
- **`mcpx configure`** — auto-generates tool documentation for CLAUDE.md from MCP server schemas. Scans configured servers and writes per-server reference files.

## [1.0.0] - 2025-03-09

### Added

- **Core CLI** — `mcpx <server> <tool> [flags]` calls any MCP tool from the terminal
- **Dynamic CLI generation** — tool flags auto-generated from MCP JSON schemas
- **Two-level config** — project (`.mcpx/config.yml`) + global (`~/.mcpx/config.yml`) with server-level merging
- **Dynamic variables** — `$(env.*)`, `$(git.*)`, `$(mcpx.*)`, `$(secret.*)`, `$(sys.*)` resolved at runtime
- **Daemon mode** — `daemon: true` keeps heavy servers alive between calls via unix socket
- **Secrets management** — `mcpx secret set/list/remove` for OS keychain integration
- **Config import** — `mcpx init` imports from `.mcp.json` (Claude Code format)
- **Health checks** — `mcpx ping <server>` verifies a server is alive with latency
- **Output modes** — `--json`, `--quiet`, `--dry-run` global flags
- **Shell completions** — `mcpx completion bash|zsh|fish|powershell` with dynamic server/tool names
- **Daemon management** — `mcpx daemon status/stop/stop-all`
- **Daemon zombie detection** — automatic shutdown when MCP server subprocess dies
