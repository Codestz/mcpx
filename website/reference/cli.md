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

Read tool arguments from stdin as a JSON object instead of flags.

```bash
echo '{"name": "Auth", "include_body": true}' | mcpx serena find_symbol --stdin
```

### `mcpx <server> --help`

Show all tools available on a server.

```bash
mcpx serena --help
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

---

## Configuration

### `mcpx init`

Import servers from `.mcp.json` (Claude Code format) into `.mcpx/config.yml`.

```bash
mcpx init
```

### `mcpx configure`

Interactive server configuration.

```bash
mcpx configure
```

### `mcpx <server> generate`

Generate a compact reference of all tools and flags. Useful for including in `CLAUDE.md`.

```bash
mcpx serena generate
mcpx serena generate --global    # write to global CLAUDE.md location
mcpx serena generate --format=md # markdown format
```

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
