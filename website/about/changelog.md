# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
