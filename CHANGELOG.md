# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
