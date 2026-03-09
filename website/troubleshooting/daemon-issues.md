# Daemon Issues

Problems with daemon mode servers.

## Stale Socket / PID File

**Symptom:** Connection refused, but `mcpx daemon status` shows nothing running.

**Cause:** The daemon crashed and left socket/PID files behind.

**Fix:**
```bash
# Clean up manually
rm /tmp/mcpx-serena-$(id -u).sock
rm /tmp/mcpx-serena-$(id -u).pid

# Next call starts a fresh daemon
mcpx ping serena
```

::: tip
As of v1.0, mcpx detects stale PID files automatically (checks if the process is alive). This issue should be rare.
:::

## Daemon Won't Start

**Symptom:** Timeout waiting for daemon.

```
error: server "serena": daemon: serena failed to start within 30s (check /tmp/mcpx-serena-501.log)
```

**Debug:**
```bash
# Check daemon logs
cat /tmp/mcpx-serena-$(id -u).log
```

Common causes:
- Server command not found (PATH issue in daemon's environment)
- Server fails to initialize (check logs for MCP handshake errors)
- Port conflict (another daemon for the same server)

## Zombie Daemon (All Requests Fail)

**Symptom:** Daemon is running but every request returns an error.

**Cause:** The MCP server subprocess crashed while the daemon was running. In versions before 1.0, the daemon would continue accepting connections but fail every request.

**Fix:** v1.0 handles this automatically — the daemon detects transport death and shuts down. If you're still seeing this:

```bash
mcpx daemon stop serena
mcpx ping serena          # starts fresh daemon
```

## Address Already in Use

```
daemon: listen /tmp/mcpx-serena-501.sock: bind: address already in use
```

**Cause:** Another daemon is using the same socket, or a stale socket file exists.

**Fix:**
```bash
mcpx daemon stop serena
rm /tmp/mcpx-serena-$(id -u).sock
```

## Daemon Logs

All daemon output goes to `/tmp/mcpx-<server>-<uid>.log`:

```bash
cat /tmp/mcpx-serena-$(id -u).log
```

Typical log entries:

```
daemon: serena listening on /tmp/mcpx-serena-501.sock (pid 12345, idle timeout 30m0s)
daemon: serena transport died, shutting down
daemon: serena exited
```

## Idle Timeout

Daemons shut down after 30 minutes of inactivity. If your server takes a long time to start and you're doing infrequent calls, you may see startup delays.

There's currently no config option to change the idle timeout — it's hardcoded at 30 minutes.

## Multiple Users

Daemon sockets include the UID: `/tmp/mcpx-serena-501.sock`. Different users on the same machine get separate daemons. Socket permissions are `0600` (owner-only).
