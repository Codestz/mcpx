# Connection Errors

Exit code 3. The server couldn't be reached.

## Command Not Found

```
error: server "serena": start: exec: "uvx": executable file not found in $PATH
```

**Cause:** The `command` in your config isn't in your PATH.

**Fix:**
1. Check the command exists: `which uvx`
2. Use an absolute path in config: `command: /usr/local/bin/uvx`
3. Ensure your PATH is set correctly in the shell mcpx runs in

::: tip
If the command works in your terminal but not from mcpx, your PATH may differ between interactive and non-interactive shells. Add the path to your `.bashrc` / `.zshrc` (not just `.bash_profile`).
:::

## Startup Timeout

```
error: server "serena": initialize: context deadline exceeded
```

**Cause:** The server took longer to start than the configured timeout (default: 30s).

**Fix:** Increase the timeout:

```yaml
servers:
  serena:
    command: uvx
    args: [...]
    startup_timeout: "120s"
```

Some servers (like Serena with LSP) need time to index. 60-120s is reasonable for first startup.

## Server Exits Immediately

```
error: server "serena": start: exit status 1
```

**Cause:** The server command exits immediately, often due to missing arguments or bad config.

**Debug:**
1. Run the command manually to see its error output:
   ```bash
   uvx --from git+https://github.com/oraios/serena serena start-mcp-server --project .
   ```
2. Use `--dry-run` to see the exact command mcpx would run:
   ```bash
   mcpx serena search_symbol --name test --dry-run
   ```
3. Check if the server expects environment variables that aren't set

## Connection Refused (Daemon)

```
error: server "serena": connect daemon: dial unix /tmp/mcpx-serena-501.sock: connection refused
```

**Cause:** The daemon's socket file exists but the process isn't listening.

**Fix:**
```bash
# Stop the stale daemon
mcpx daemon stop serena

# Or manually clean up
rm /tmp/mcpx-serena-$(id -u).sock
rm /tmp/mcpx-serena-$(id -u).pid

# Next call will start a fresh daemon
mcpx ping serena
```

## Permission Denied

```
error: server "serena": start: fork/exec /usr/local/bin/uvx: permission denied
```

**Cause:** The command binary doesn't have execute permission.

**Fix:**
```bash
chmod +x /usr/local/bin/uvx
```

## Wrong Protocol

If the server expects SSE but config says stdio (or vice versa), you'll see garbled output or immediate disconnection.

**Fix:** Ensure `transport` matches what the server expects:

```yaml
transport: stdio     # for servers that communicate via stdin/stdout
```
