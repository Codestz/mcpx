# Troubleshooting

Common issues and how to fix them.

## Quick Diagnosis

```bash
# Check mcpx is installed
mcpx version

# Check config is valid
mcpx list

# Check a server is reachable
mcpx ping serena

# Check daemon status
mcpx daemon status

# See what would execute (without executing)
mcpx serena search_symbol --name "test" --dry-run

# Check daemon logs
cat /tmp/mcpx-serena-$(id -u).log
```

## Common Issues

### Server won't start

```
error: server "serena": start: exec: "uvx": executable file not found in $PATH
```

The server command isn't in your PATH. See [Connection Errors](/troubleshooting/connection-errors).

### Config not found

```
No servers configured.
Add servers to .mcpx/config.yml or ~/.mcpx/config.yml
```

Create a config file or run `mcpx init` to import from `.mcp.json`. See [Config Errors](/troubleshooting/config-errors).

### Daemon is stale

```
error: server "serena": daemon: connect: dial unix /tmp/mcpx-serena-501.sock: connection refused
```

The daemon died but left files behind. See [Daemon Issues](/troubleshooting/daemon-issues).

### Secret not found

```
error: server "serena": resolve env: secret get "github_token": secret not found
```

The secret isn't in your keychain. See [Secrets Errors](/troubleshooting/secrets-errors).

### macOS keychain popup

macOS may prompt for keychain access when mcpx reads secrets. See [Platform-Specific](/troubleshooting/platform-specific).

## By Topic

- [Connection Errors](/troubleshooting/connection-errors) — server won't start, timeout, wrong command
- [Daemon Issues](/troubleshooting/daemon-issues) — stale sockets, zombies, logs
- [Secrets Errors](/troubleshooting/secrets-errors) — keychain access, missing secrets
- [Config Errors](/troubleshooting/config-errors) — YAML issues, variable resolution, merge confusion
- [Platform-Specific](/troubleshooting/platform-specific) — macOS, Linux, Windows/WSL quirks
