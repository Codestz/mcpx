# mcpx Roadmap

> What's next for mcpx — focused on impact, not infrastructure.

mcpx v1.0.0 shipped with the core adapter, daemon mode, secrets, and shell completions. Everything below builds on that foundation to make the tool sharper for the people already using it.

---

## v1.1 — Gain & Fast Pipes

**Theme:** Prove the savings. Make pipes real.

mcpx claims to save tokens — `mcpx gain` shows the receipts. And pipes that work in a demo should work at scale.

### `mcpx gain` — Token savings dashboard

Passively tracks every call and schema size. Zero overhead — just appends a log line.

```
$ mcpx gain

  mcpx token savings
  ──────────────────────────────────────

  Today         4,218 tokens saved    (12 calls)
  This week    38,664 tokens saved    (147 calls)
  All time    284,910 tokens saved    (1,203 calls)

  Server breakdown
  ──────────────────────────────────────
  serena        2,148 tok/call × 1,089 calls = 233,972 saved
  seq-thinking    412 tok/call ×   114 calls =  46,968 saved

  Performance
  ──────────────────────────────────────
  Avg latency       142ms (daemon)  vs  ~1,200ms (cold start)
  Daemon hit rate    94% (1,131 / 1,203)
  Time saved         ~19 min from daemon reuse
```

- `mcpx gain --json` — structured output for dashboards
- `mcpx gain --history` — daily/weekly trend
- `mcpx gain --server serena` — filter by server

### Fast pipes

- **Schema caching** — cache `tools/list` locally, skip MCP negotiation on repeat calls. Critical for xargs/loops where mcpx runs 20+ times.
- **Connection reuse** — daemon calls batch over the same unix socket. Zero cold start per invocation.
- **`--flag -` reads from stdin** — `jq '.path' | mcpx serena get_symbols_overview --relative_path -`. Single flag from pipe, not full JSON mode.

---

## v1.2 — Config Quality of Life

**Theme:** Remove paper cuts.

- **`mcpx sync`** — bidirectional sync between `.mcpx/config.yml` and `.mcp.json`. Stop maintaining both manually.
- **`mcpx doctor`** — diagnose common issues: server won't start, bad config paths, stale daemons, missing binaries. One command to answer "why isn't this working?"
- **Config validation** — catch typos, bad schemas, unreachable servers before runtime. Clear errors with line numbers.

---

## v1.3 — Output Control

**Theme:** Skip jq for the 80% case.

- **`--template`** — Go template strings on output: `--template "{{len .results}} found"`.
- **`--field`** — extract a single field: `--field "results[0].path"`. Covers most simple extractions without jq.
- **Structured exit codes** — distinct codes for tool error (1), config error (2), connection error (3), timeout (4). Scripts need this to branch correctly.

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
