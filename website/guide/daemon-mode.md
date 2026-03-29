# Daemon Mode

Daemon mode keeps MCP servers alive between calls. Instead of spawning a new process for every command, mcpx connects to a running server via unix socket.

## When to Use It

Use daemon mode for servers that:
- Have slow startup (LSP initialization, loading indexes)
- Are called frequently in a session
- Maintain state between calls

Don't bother for servers that start instantly and are called rarely.

## Enabling Daemon Mode

Set `daemon: true` in your server config:

```yaml
servers:
  serena:
    command: uvx
    args: [...]
    daemon: true
```

mcpx automatically starts the daemon on first use and connects via socket on subsequent calls.

## How It Works

1. First call to `mcpx serena <tool>`: spawns a detached daemon process
2. Daemon starts the MCP server, performs the handshake, caches the server's capabilities
3. `mcpx` connects to the socket, replays the cached handshake, sends the request, gets the response
4. CLI exits. Daemon stays alive.
5. Next call: connects to existing socket — zero startup cost

The daemon caches the `InitializeResult` from the MCP handshake and replays it to each connecting client. This means `mcpx <server> info`, `--help` (with prompts/resources), and all capability-based features work correctly in daemon mode.

```
mcpx CLI ──► unix socket ──► daemon process ──► MCP server subprocess
                                (stays alive)     (stays alive)
```

### Socket Location

Daemons are scoped by project and workspace to prevent cross-session conflicts:

```
/tmp/mcpx-<server>-<scope>-<uid>.sock    # unix socket (mode 0600)
/tmp/mcpx-<server>-<scope>-<uid>.pid     # PID file
/tmp/mcpx-<server>-<scope>-<uid>.log     # daemon log
```

The `<scope>` is a short hash (8 hex chars) of the project root path + workspace name. This means:

- **Two different projects** → two separate daemons (different scope hashes)
- **Two workspaces in the same monorepo** → two separate daemons
- **Same project, same workspace** → shared daemon (correct)

This prevents a common problem: two Claude Code sessions working on different projects would share one Serena daemon, racing on `activate_project` and corrupting each other's context.

```bash
mcpx daemon status
#   serena (a1b2c3d4)  running  /tmp/mcpx-serena-a1b2c3d4-501.sock
#   serena (e5f6g7h8)  running  /tmp/mcpx-serena-e5f6g7h8-501.sock
```

## Managing Daemons

### Check status

```bash
mcpx daemon status
#   serena  running  /tmp/mcpx-serena-501.sock
```

### Stop a daemon

```bash
mcpx daemon stop serena
# Stopped daemon for serena
```

### Stop all daemons

```bash
mcpx daemon stop-all
```

### Health check

```bash
mcpx ping serena
# serena: ok (21 tools, 47ms)
```

## Idle Timeout

Daemons shut down after 30 minutes of inactivity by default. Every request resets the timer.

## Crash Recovery

If the MCP server subprocess crashes while the daemon is running:

1. The daemon detects the transport death (stdout pipe closes)
2. Active connections receive an error response
3. The daemon shuts down and cleans up its socket and PID file
4. The next `mcpx` call starts a fresh daemon automatically

This prevents "zombie daemons" — processes that accept connections but fail every request.

## Logs

Daemon logs are written to `/tmp/mcpx-<server>-<uid>.log`:

```bash
cat /tmp/mcpx-serena-$(id -u).log
```

```
daemon: serena listening on /tmp/mcpx-serena-501.sock (pid 12345, idle timeout 30m0s)
daemon: serena idle for 30m0s, shutting down
daemon: serena exited
```

## Troubleshooting

See [Daemon Issues](/troubleshooting/daemon-issues) for common problems.
