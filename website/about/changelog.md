# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
