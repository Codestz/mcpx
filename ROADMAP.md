# mcpx Roadmap

> What's next for mcpx — focused on impact, not infrastructure.

mcpx v1.1.0 added ergonomic enhancements (`@file`, `--pick`, `--timeout`, stdin merge, `configure`). Everything below builds on that foundation to make the tool sharper for the people already using it.

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

## v1.3 — Output Control

**Theme:** Advanced output formatting.

- **`--template`** — Go template strings on output: `--template "{{len .results}} found"`.
- **Structured exit codes** — distinct code for timeout (4), in addition to tool error (1), config error (2), connection error (3). Scripts need this to branch correctly.

> Note: Basic field extraction is covered by `--pick` (shipped in v1.1).

---

## v1.4 — Multi-Server Workflows

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

- **`mcpx install <server>`** — install MCP servers from a registry
- **`mcpx registry search`** — discover servers by capability
- **Config sharing** — shareable `.mcpx/config.yml` snippets
- **Plugin system** — custom variable resolvers

---

## Principles

1. **Ship what matters now.** No ecosystem plays until the core is razor sharp.
2. **Each version is independently useful.** No version depends on the next.
3. **Measure before optimizing.** `mcpx gain` ships first because you can't improve what you can't see.
4. **UNIX philosophy.** Every feature composes with pipes, scripts, and existing tools.
