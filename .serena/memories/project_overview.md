# mcpx - MCP Server CLI Proxy

## Purpose
mcpx wraps MCP (Model Context Protocol) servers into CLI tools so AI agents call them via Bash instead of loading tool schemas into context. Eliminates 50-100K tokens of context overhead per session.

## Tech Stack
- Go 1.22, Cobra (CLI), yaml.v3 (config), go-keyring (secrets), fatih/color (output)
- Single binary, zero runtime dependencies

## Architecture
- cmd/mcpx/main.go - entrypoint
- internal/config/ - two-level config (global ~/.mcpx + project .mcpx), YAML, merging
- internal/resolver/ - dynamic variables: env, git, mcpx, secret, sys namespaces
- internal/mcp/ - MCP protocol client, JSON-RPC 2.0 over stdio, Transport interface
- internal/cli/ - Cobra commands, dynamic tool dispatch from MCP schemas, output formatting
- internal/daemon/ - daemon process (keeps servers warm via unix socket, ~70ms per call)
- internal/secret/ - OS keychain integration via go-keyring

## Package Dependency Graph (acyclic)
- config, secret: leaf packages (no internal deps)
- resolver: depends on secret
- mcp: zero internal deps (receives resolved strings)
- daemon: depends on mcp
- cli: orchestrator, wires everything together

## Key Design
- AI-first: designed for AI agents calling tools through Bash
- On-demand discovery: mcpx list instead of upfront schema loading
- UNIX composability: stdin/stdout/stderr, pipes, exit codes
- Two-level config: project overrides global
- Daemon mode: keeps heavy servers warm, auto-exits after 30min idle