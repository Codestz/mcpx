# mcpx v1.6 — Agentic Supremacy

> The thesis: mcpx is not just a CLI proxy — it's the intelligent control plane between AI agents and MCP servers. v1.6 ships the data layer, the agent superpowers, and the always-on dashboard that prove the value.

**Three goals every feature must serve:**
1. **Less tokens** — schema cache, result cache, `find`, compact-default help, response truncation
2. **More accuracy** — structured exit codes, schema normalization, `--example`, error remediation
3. **More control** — gain analytics, dashboard, audit, agent identity

---

## Build sequence

The order matters. Each layer's data is consumed by the next:

```
   ┌─ schema cache ──┐         ┌─ mcpx find ─┐
   │                 ├─→ JSONL ┤              ├─→ mcpx gain (TUI)
   │  result cache ──┤  stats  │  mcpx batch ─┘
   │                 │         │
   │  config.gain ───┘         └────────────────→ dashboard (always-on)
```

**Foundation (must ship first):**
- F1. JSONL stats schema + writer (the data contract everyone consumes)
- F2. Schema cache (server init/tools/prompts/resources)
- F3. Config: `gain:`, `cache:`, `ui:` sections
- F4. Instrumentation: hook every tool call to emit a stats record

**Agent superpowers:**
- A1. `mcpx find` — semantic tool search across all servers
- A2. Result cache for idempotent reads
- A3. `mcpx batch` — NDJSON in/out, parallel exec
- A4. Default-compact help; verbose behind `--full`
- A5. `--example` + arg validation

**Operator surfaces:**
- O1. `mcpx gain` — premium TUI with sparklines, top-K tables, savings front-and-center
- O2. Always-on dashboard daemon (auto-spawned, 127.0.0.1, single page)
- O3. Project-aware views (sidebar, per-project filtering)

**Robustness:**
- R1. Issue #14 + canonical schema normalizer (oneOf/anyOf/allOf/$ref/union types)
- R2. Structured exit codes (0–6)
- R3. Active error remediation (typo suggestions on tool/flag mismatches)

---

## F1. JSONL stats — the data contract

**Path:** `~/.mcpx/stats.jsonl` (single file, append-only). Daily rotation handled at read time, not write time, to avoid coordination.

**Per-call schema** (one line per call, ~250 bytes):

```json
{
  "ts": "2026-05-03T14:12:33.482Z",
  "session": "9214",
  "project": "/Users/x/projects/mcpx",
  "agent": "claude-code",
  "server": "serena",
  "tool": "find_symbol",
  "args_bytes": 84,
  "args_tokens_est": 21,
  "response_bytes": 4210,
  "response_tokens_est": 1052,
  "latency_ms": 47,
  "transport": "stdio",
  "daemon": true,
  "schema_cache_hit": true,
  "result_cache_hit": false,
  "exit_code": 0,
  "error": null,
  "policy_action": "allow",
  "policy_name": null,
  "native_baseline_tokens": 8421,
  "tokens_saved": 7369
}
```

**Fields explained:**
- `session` = PPID of the caller (groups calls within one shell session).
- `project` = resolved project root (or `""`). Drives dashboard sidebar.
- `agent` = `MCPX_AGENT` env, else `"unknown"`. Enables multi-agent identity later.
- `*_tokens_est` = `bytes / 4`. Honest estimate. tiktoken upgrade in v1.6.1.
- `native_baseline_tokens` = JSON-stringified size of `(initialize result + tools/list + prompts/list + resources/list)` for this server, computed once when schema cache populates.
- `tokens_saved` = `native_baseline_tokens - args_tokens_est - response_tokens_est`. The headline metric.

**Writer guarantees:**
- Atomic single-line append (`O_APPEND` + write < PIPE_BUF).
- Never blocks the caller: writer runs in a goroutine with bounded buffer; drops on overflow with a counter.
- Never errors out the call. Stats failure is invisible to the user.

**Package:** `internal/stats/`
- `stats.go` — `Record` struct, `Writer.Write(Record)`, `Writer.Flush()`.
- `read.go` — `Reader.Iter(filter)`, time-range filtering, project filter, server/tool filter.
- `agg.go` — `Aggregate(filter)`: top tools, latency p50/p95, saved tokens, hit rates, daily buckets.

---

## F2. Schema cache

**Path:** `~/.mcpx/cache/<server-hash>.json`

`server-hash` = SHA256 of `(command, args, env, url)`. Different config = different cache entry, no false hits.

**Cache record:**
```json
{
  "version": 1,
  "captured_at": "...",
  "ttl_seconds": 300,
  "server_hash": "abc123...",
  "initialize": { ... },
  "tools": [ ... ],
  "prompts": [ ... ],
  "resources": [ ... ],
  "native_baseline_tokens": 8421
}
```

**Behavior:**
- TTL: default 5 min (config: `cache.schema_ttl: 5m` per server).
- Bypass: `--no-cache` flag, `MCPX_CACHE=off` env.
- Invalidate: `mcpx cache clear [server]`, auto on `notifications/tools_list_changed`.
- Daemons keep the schema in memory; the cache is for non-daemon and CLI-mode invocations.

**Package:** `internal/schemacache/` (new — separate from `internal/cache/` which is empty).

---

## F3. Config additions

```yaml
# ~/.mcpx/config.yml
gain:
  enabled: true              # default: true
  tokenizer: estimate        # estimate | tiktoken (v1.6.1)
  retain_days: 30            # auto-prune stats.jsonl entries older than N days

cache:
  schema_ttl: 5m
  result_ttl: 30s
  result_enabled: true       # only idempotent tools cached

ui:
  enabled: true              # default: true. set false to disable always-on dashboard
  port: 7878                 # 0 = random
  bind: 127.0.0.1
  idle_timeout: 1h
```

Per-server overrides for `cache:` allowed under each `servers.<name>:`.

---

## F4. Instrumentation

Single hook in `internal/cli/commands.go` `runTool`:

```go
defer func() { stats.Record(buildRecord(...)) }()
```

Captures: timing (start → defer), bytes in (args JSON), bytes out (response JSON), cache hits, policy decision, error/exit code. Zero overhead when `gain.enabled: false`.

Schema cache populates `native_baseline_tokens` on first server connect; instrumented call reads it from the cache.

---

## A1. `mcpx find`

**Surface:**
```bash
mcpx find "search code by regex"
mcpx find "issue tracker" --top 3
mcpx find "..." --json                 # machine-readable
mcpx find "..." --server serena        # restrict to one server
```

**Output (default, ~80 tokens):**
```
serena.search_for_pattern   0.91  Flexible regex search across files
serena.find_symbol          0.74  Retrieve info on symbols by name path
github.search_issues        0.42  Search GitHub issues by query
```

**Algorithm:**
- Build corpus = `tool.name + " " + tool.description` for every tool of every configured server (read from schema cache; populate cache for any server not seen).
- Score = BM25 over query tokens vs. corpus. Bonus weight if query token appears in `tool.name`.
- Snake-case and camelCase splitter for token matches (`find_symbol` → `find symbol` → matches "symbol").
- Top-K (default 5).

**Effort:** ~250 LoC, no new deps. Package: `internal/find/`.

---

## A2. Result cache

Only for tools annotated as idempotent. Detection (in priority order):
1. Server-supplied `annotations.readOnlyHint: true` (per MCP 2025 spec).
2. Config-level: `cache.result_idempotent_tools: ["serena.find_symbol", ...]`.
3. Heuristic: tool name starts with `get_`, `list_`, `find_`, `search_`, `read_` (opt-in via `cache.result_heuristic: true`).

**Key:** `SHA256(server, tool, normalized-args-JSON)`.
**Path:** `~/.mcpx/cache/results/<key>.json`.
**TTL:** `cache.result_ttl` (default 30s; conservative).
**Stats:** every cache hit emits a stats record with `latency_ms` near zero and `result_cache_hit: true`.

---

## A3. `mcpx batch`

**Input (NDJSON via stdin):**
```jsonl
{"id":"a","server":"serena","tool":"find_symbol","args":{"name_path_pattern":"Auth"}}
{"id":"b","server":"serena","tool":"find_symbol","args":{"name_path_pattern":"Token"}}
{"id":"c","server":"github","tool":"get_issue","args":{"number":42}}
```

**Output (NDJSON, same order):**
```jsonl
{"id":"a","ok":true,"latency_ms":42,"result":{...},"cached":false}
{"id":"b","ok":true,"latency_ms":2,"result":{...},"cached":true}
{"id":"c","ok":false,"latency_ms":120,"error":"...","exit_code":1}
```

**Flags:**
- `--parallel` (default for daemon-backed servers; bounded by `--max-concurrent N`)
- `--sequential`
- `--max-concurrent N` (default = NumCPU)
- `--stop-on-error`
- `--cache` (use result cache; default true)

**Per-server connection reuse:** one client per server name across the batch; closed at end.

**Effort:** ~400 LoC. Package: `internal/cli/batch.go`.

---

## A4. Default-compact help

Flip the defaults:
- `mcpx <server> --help` and `mcpx list <server>` → one line per tool: `name  required-args-summary`. No descriptions.
- `mcpx <server> --help --full` and `mcpx list <server> -v` → current verbose output.

This is a UX change. Current `printToolsVerbose` becomes opt-in. New `printToolsCompact` is the default.

---

## A5. `--example` + arg validation

```bash
mcpx serena find_symbol --example
# {
#   "name_path_pattern": "<string>",
#   "relative_path": "<string>",
#   "depth": 0,
#   "include_body": false
# }
```

`--validate-args` checks types and required fields without calling the tool. Builds on schema normalizer (R1).

---

## O1. `mcpx gain` — premium terminal UI

**Goal:** screenshot-worthy. Not a flat table.

**Default view (`mcpx gain`):**
```
┌─ mcpx ───────────────────────── this project · 7d ──┐
│                                                       │
│    Tokens saved        487,231                        │
│    ▔▔▔▔▔▔▔▔▔▔▔▔▔     ▁▂▃▅▇▆▄▂▃ daily                 │
│                                                       │
│    Calls          1,284      Cache hit rate    73%   │
│    Avg latency      41ms     Errors              2%  │
│                                                       │
├─ Top tools ─────────────────────────────────────────┤
│  serena.find_symbol           512   ▓▓▓▓▓▓▓▓▓        │
│  serena.search_for_pattern    284   ▓▓▓▓▓            │
│  github.get_issue             107   ▓▓                │
│  serena.get_symbols_overview   89   ▓▓                │
│  ...                                                  │
├─ Top savings ───────────────────────────────────────┤
│  serena.find_symbol     192K saved  (vs native MCP)  │
│  github.get_issue        61K saved                    │
├─ Last 5 calls ──────────────────────────────────────┤
│  14:23  serena.find_symbol      47ms  ✓              │
│  14:23  serena.find_symbol     cached ⚡              │
│  14:22  github.get_issue       180ms  ✓              │
│  ...                                                  │
└────────────────────────────── http://127.0.0.1:7878 ┘
```

**Renderer requirements:**
- Use box-drawing chars + `fatih/color` (already a dep). No new deps.
- Width detection via `term.GetSize`. Falls back to 80 cols.
- Sparklines: 8-level Unicode block chars (`▁▂▃▄▅▆▇█`).
- Bars: 8-step block chars for fractional widths.
- Number formatting: `1.2K`, `487K`, `4.8M`.
- Colors: green for saved/ok, yellow for warn, red for errors, dim for metadata.

**Subcommands:**
```bash
mcpx gain                          # the dashboard above (current project, 7d)
mcpx gain --all                    # all projects combined
mcpx gain --project /path          # specific project
mcpx gain --since 24h              # time window
mcpx gain --by tool                # ranked tool table only
mcpx gain --by server              # ranked server table
mcpx gain --by day                 # daily sparkline detail
mcpx gain --history [N]            # last N call entries
mcpx gain --suggest                # mined recommendations
mcpx gain --json                   # machine-readable everything
mcpx gain --watch                  # live refresh every 1s
```

**Package:** `internal/cli/gain.go` + `internal/render/` (new helpers: `Box`, `Bar`, `Sparkline`, `FormatNumber`, `FormatBytes`, `FormatDuration`).

---

## O2 + O3. Always-on dashboard

**Lifecycle:** lazy supervisor daemon, auto-spawned on first `mcpx <command>` invocation.

**Spawn flow:**
1. Every CLI startup checks `~/.mcpx/ui.json`. If file missing or PID dead → spawn UI daemon.
2. UI daemon writes `{port, token, pid}` (mode 0600) to `~/.mcpx/ui.json`.
3. CLI prints one-line stderr notice **once per shell session** (suppression via `MCPX_UI_NOTICE_SHOWN` env that the CLI sets before it execs anything).
4. Idle timeout: 1h with no HTTP traffic → daemon self-exits.
5. Opt-out: `ui.enabled: false` in config or `MCPX_UI=off` env.
6. Manual: `mcpx ui status | stop | open | disable`.

**Architecture:**
- New package `internal/ui/`:
  - `supervisor.go` — `EnsureRunning()`, `~/.mcpx/ui.json` handshake.
  - `server.go` — `http.Server`, SSE `/events`, JSON API, static `embed.FS`.
  - `data.go` — wraps `internal/stats` reader + cache for query endpoints.
  - `assets/` — single-page HTML, CSS, htmx, uPlot. ~50KB total embedded.
- New hidden cobra command in `internal/cli/`: `mcpx __ui` (mirrors `__daemon` pattern).
- Hook into `cli.Execute()` startup: `ui.EnsureRunningAsync()` (non-blocking).

**Single-page layout (HTML/CSS):**
```
┌──────────────────────────────────────────────────────────┐
│ mcpx                              live ●   saved: 487K    │
├───────────────┬──────────────────────────────────────────┤
│ PROJECTS      │ Project: mcpx                             │
│ ▸ mcpx (this) │ ─────────────────────────────────────     │
│   my-app      │  ╭─ 7-day savings ──╮  ╭─ Hit rate ──╮   │
│   work-stuff  │  │  ▁▂▃▄▆▇▅▃        │  │   73%       │   │
│   All         │  │  ▔▔▔▔▔▔▔▔▔▔      │  ╰─────────────╯   │
│               │  ╰──────────────────╯                     │
│ SERVERS       │                                            │
│ ▸ serena (✓)  │  Top tools (7d) ────────────              │
│   github      │   serena.find_symbol     ▓▓▓▓▓▓▓▓ 512    │
│   sentry      │   serena.search_pattern  ▓▓▓▓     284    │
│               │   github.get_issue       ▓▓       107    │
│ AUDIT         │                                            │
│ Live tail     │  Live tail (SSE) ──────────                │
│               │   14:23 serena.find_symbol  47ms ✓        │
│               │   14:23 serena.find_symbol  cached ⚡      │
│               │   14:22 github.get_issue   180ms ✓        │
│               │   ...                                      │
│               │                                            │
│               │  [ Replay last call ]  [ Export JSON ]    │
└───────────────┴──────────────────────────────────────────┘
```

**Tech stack (zero npm):**
- Go server: stdlib `net/http` + `html/template` + SSE.
- Frontend: htmx (15KB) + uPlot (45KB) + handcrafted CSS.
- All assets bundled via `embed.FS`.
- Token-protected URLs: every page/API call requires `?t=<token>` from `~/.mcpx/ui.json`.

**Pages (server-rendered + htmx swaps):**
- `/` — overview (this project, default)
- `/project/<slug>` — same layout, scoped
- `/all` — cross-project totals
- `/tools` — full ranked table (sortable by frequency, latency, savings, errors)
- `/servers` — health, daemons, schema age
- `/audit` — denied calls, policy decisions
- `/api/events` — SSE stream (live tail)
- `/api/stats?...` — JSON
- `/api/replay/<id>` — re-run a past call

---

## R1. Schema normalizer (issue #14 + beyond)

`internal/mcp/schema.go` (new). One pass over server-provided JSON Schema that:
- Accepts `type` as string OR `[]string` → picks first non-null, sets `Nullable: true` if null was in the union.
- Resolves `$ref` inline (within same schema document).
- Flattens `allOf` by merging subschemas.
- Picks first non-null branch of `oneOf` / `anyOf`, stores alternatives in `Ext`.
- Preserves unknown keywords in `Ext map[string]json.RawMessage`.
- Never errors. Logs warnings under `MCPX_VERBOSE=1`.

Result: every downstream consumer (`describe`, `--help`, `--example`, validation, dashboard) sees a single canonical shape.

---

## R2. Structured exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Tool error (`isError=true`) |
| 2 | Config / syntax error |
| 3 | Connection / transport error |
| 4 | Timeout |
| 5 | Policy denied |
| 6 | Tool not found |

Exposed via `cli.Exit()` helper called from every command. README + `--help` document them.

---

## R3. Active error remediation

When tool/flag lookup fails:
- "tool `find_symbl` not found in `serena`. **Did you mean `find_symbol`?**" — Levenshtein ≤ 2.
- "flag `--namepath` not recognized. **Did you mean `--name_path_pattern`?**"
- "required flag `--query` missing. **Example:** `mcpx serena search_for_pattern --query 'TODO'`" — uses `--example` machinery.

---

## Implementation order (locked)

| # | Item | Effort | Lands as |
|---|------|--------|----------|
| 1 | F3 config additions | XS | infra |
| 2 | F1 stats package + JSONL writer | S | infra |
| 3 | F4 instrumentation hook in `runTool` | S | infra |
| 4 | F2 schema cache | M | infra |
| 5 | R1 schema normalizer + issue #14 fix | M | robustness |
| 6 | R2 structured exit codes | XS | robustness |
| 7 | A1 `mcpx find` | M | agent |
| 8 | A2 result cache | S | agent |
| 9 | A3 `mcpx batch` | M | agent |
| 10 | A4 default-compact help | XS | agent |
| 11 | A5 `--example` + validate | S | agent |
| 12 | R3 error remediation | S | polish |
| 13 | O1 `mcpx gain` premium TUI + `internal/render` | L | operator |
| 14 | O2 UI daemon supervisor + lazy spawn | M | operator |
| 15 | O3 dashboard HTML/SSE/static assets | L | operator |
| 16 | tiktoken-go via `+tiktoken` build tag | S | v1.6.1 |

XS = <0.5d, S = 1d, M = 2–4d, L = 5–10d.

---

## Validation strategy

- Each package ships with table-driven Go tests.
- Local install via `make install`; replace `~/go/bin/mcpx` after every milestone.
- Dogfood: every code change in v1.6 development uses the WIP mcpx via serena. Friction points become the next iteration's tasks.
- Keep `mcpx ping serena` green at every milestone (smoke test).
- Add `make e2e` that runs `mcpx serena ping` + `mcpx find ...` + `mcpx gain --json` against a known-good config.

---

## Out of scope for v1.6

Defer to v1.7+:
- Rate limiting enforcement
- Output filtering / redaction
- Multi-agent identity beyond `MCPX_AGENT` env
- `mcpx sync` (.mcp.json bidirectional)
- Notification handler (real-time MCP events)
- Docker runtime
- `mcpx install <server>` registry
- tiktoken-go via build tag

These are good ideas but adding them dilutes v1.6's coherent story.

---

## v1.6 final scope (shipped)

**Foundation**
- F1 stats package (writer, reader, aggregator, ID generator, result preview cap)
- F2 schema cache + result cache + idempotence detection
- F3 config (gain/cache/ui sections + Default helpers)
- F4 instrumentation (every tool call writes a JSONL record with full args + truncated result preview)

**Robustness**
- R1 schema normalizer (issue #14: type union arrays + oneOf/anyOf/allOf/Ext)
- R2 structured exit codes (0–6)
- R3 typo remediation (Levenshtein on tool names + flag names)

**Agent superpowers**
- A1 `mcpx find` — BM25 ranked tool discovery
- A2 result cache (heuristic + allow-list)
- A3 `mcpx batch` — NDJSON in/out, parallel by default, client pool
- A4 default-compact help (verbose behind `--full`)
- A5 `--example` (JSON skeleton from normalized schema) + `--validate-args`

**Operator surfaces**
- O1 `mcpx gain` premium TUI: hero metric, sparkline, top-tools bars, top-savers, server p95, last calls, dashboard URL footer; subcommands `--by tool|server|day`, `--history`, `--suggest`, `--watch`, `--json`, `--all`, `--project`, `--since`
- O2 always-on dashboard daemon (lazy spawn, token-protected, idle-shutdown 1h, `MCPX_UI=off` opt-out, `mcpx ui status|stop|open|disable`)
- O3 redesigned single-page dashboard: project sidebar, time range tabs (1h/24h/7d/30d/all), hero metric (64px), token efficiency bar, 4 stat tiles, top-tools table (clickable for drill-down), top-savers, per-server health cards (status/calls/p95/err), live tail with regex filter (`/`), click-to-inspect drawer (full args + result preview + replay button), SSE connection indicator, freshness label

**Operational**
- `mcpx doctor` — config, command, daemon, initialize, tools/list, secret resolution checks; `--json` mode
- Dirty-build warning (`mcpx version` flags `+dirty`)
- 276 tests across 13 packages; `go vet` clean

**Friction surfaced via dogfooding** (in `.dogfood/v16-friction.md`)
- serena `replace_symbol_body` corrupts file when body includes leading keyword
- Stale LSP diagnostics persist across edits
- /tmp scratch files trigger phantom diagnostics
- Dirty builds silently propagate stale debug code
- Notice gating per-command (resolved)
- Memory file index format violation (resolved)
