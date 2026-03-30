# mcpx Roadmap

> What's next for mcpx — focused on impact, not infrastructure.

mcpx v1.5.0 transforms mcpx from a CLI proxy into a secure gateway for MCP servers. Security policies, audit logging, and scoped daemons make mcpx ready for teams.

---

## v1.5 — Secure Gateway ✅ (shipped 2026-03-30)

**Theme:** The missing control plane between AI agents and MCP servers.

### Shipped

- **Security policy engine** — tool allow/deny, argument inspection, content regex (SQL mutations, path traversal). Global + per-server cascading.
- **Security modes** — `read-only`, `editing`, `custom` presets. Per-server.
- **Audit logging** — JSONL audit trail for every tool call. Secret redaction. Configurable log path.
- **Scoped daemons** — each project gets its own daemon process, preventing cross-session races.
- **Allowed/blocked tool lists** — per-server whitelist/blacklist with glob patterns.
- **Content matching** — `deny_pattern`, `require_pattern`, `when` clause for deep argument inspection.
- **Website redesign** — new landing page, Security section (5 pages), Integrations section (5 pages).
- **README rebrand** — "Secure gateway for MCP servers — from CLI to production."

---

## v1.1 — Ergonomic Enhancements ✅ (shipped 2026-03-13)

**Theme:** Make mcpx natural for AI agents and human operators.

### Shipped

- **`@file` syntax** — any string flag accepts `@/path` to read from file or `@-` for stdin. No more shell escaping.
- **`--pick`** — extract JSON fields from output without jq (e.g. `--pick 0.name`).
- **`--timeout`** — per-call timeout override (e.g. `--timeout 60s`).
- **stdin merge** — `--stdin` now merges with CLI flags (flags win on conflict).
- **`mcpx configure`** — auto-generate tool docs for CLAUDE.md from MCP server schemas.

### Deferred to later

- **`mcpx gain`** — token savings dashboard. Moved to v1.2.
- **Schema caching** — cache `tools/list` locally. Moved to v1.2.

---

## v1.2 — Gain, Caching & Config QoL

**Theme:** Prove the savings. Remove paper cuts.

- **`mcpx gain`** — token savings dashboard. Passively tracks every call and schema size. `mcpx gain --json`, `--history`, `--server`.
- **Schema caching** — cache `tools/list` locally, skip MCP negotiation on repeat calls. Critical for xargs/loops.
- **`mcpx sync`** — bidirectional sync between `.mcpx/config.yml` and `.mcp.json`. Stop maintaining both manually.
- **`mcpx doctor`** — diagnose common issues: server won't start, bad config paths, stale daemons, missing binaries. One command to answer "why isn't this working?"
- **Config validation** — catch typos, bad schemas, unreachable servers before runtime. Clear errors with line numbers.

---

## v1.3 — Full MCP Coverage ✅ (shipped 2026-03-14)

**Theme:** Complete MCP spec support — tools, prompts, and resources.

### Shipped

- **Prompts** — `prompt list`, `prompt <name> --help`, `prompt <name> [--arg val ...]`. Full `prompts/list` and `prompts/get` with pagination.
- **Resources** — `resource list`, `resource read <uri>`. Full `resources/list`, `resources/templates/list`, and `resources/read` with pagination.
- **Server info** — `mcpx <server> info` shows server name, version, protocol version, and capability checklist.
- **Daemon capability forwarding** — daemon caches `InitializeResult` and replays it to clients. All features now work in daemon mode.
- **Prompt argument validation** — required arguments checked before RPC call.
- **Enhanced server help** — `--help` shows prompts and resources alongside tools.
- **Generate v2** — `configure` and `generate` include prompts/resources in generated docs.

### Deferred to later

- **`--template`** — Go template strings on output. Moved to v1.4.
- **Structured exit codes** — distinct timeout code (4). Moved to v1.4.

---

## v1.4 — Output Control & Multi-Server Workflows

**Theme:** Repeatable pipelines.

- **`mcpx run`** — multi-step pipelines defined in config:
  ```yaml
  workflows:
    audit-auth:
      - serena find_symbol --name_path_pattern "Auth"
      - serena search_for_pattern --substring_pattern "TODO" --relative_path auth/
  ```
- **Cross-server pipes** — output of one server feeds into another. The plumbing works via stdout; `mcpx run` makes it declarative and shareable.
- **`mcpx run --parallel`** — run independent steps concurrently.

---

## v1.5 — Observability

**Theme:** Debug failures, understand performance.

The stats collection from `mcpx gain` (v1.1) already captures the data. This release adds more views.

- **`mcpx logs`** — structured log of recent calls: request, response, latency. Stored in `~/.mcpx/logs/`.
- **`--timing`** — execution breakdown on any call: connection, negotiation, execution, total.
- **`--verbose`** — raw JSON-RPC request/response. The `curl -v` of MCP.

---

## Future — Ecosystem (v2.0+)

Not committed to a timeline. Exploring:

- **Docker/Podman runtime** — `runtime: docker` in config, run MCP servers in isolated containers
- **Multi-agent identity** — track which agent made each call in audit logs
- **Rate limiting enforcement** — enforce `max_calls_per_minute` at runtime
- **`mcpx install <server>`** — install MCP servers from a registry
- **`mcpx registry search`** — discover servers by capability
- **Config sharing** — shareable `.mcpx/config.yml` snippets
- **Plugin system** — custom variable resolvers, custom policy evaluators

---

## Principles

1. **Ship what matters now.** No ecosystem plays until the core is razor sharp.
2. **Each version is independently useful.** No version depends on the next.
3. **Measure before optimizing.** `mcpx gain` ships first because you can't improve what you can't see.
4. **UNIX philosophy.** Every feature composes with pipes, scripts, and existing tools.
