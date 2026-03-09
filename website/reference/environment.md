# Environment

Environment variables and paths that mcpx respects.

## Environment Variables

| Variable | Effect |
|----------|--------|
| `NO_COLOR` | Disables colored output (any value). Follows [no-color.org](https://no-color.org) |
| `HOME` | Used for `~/.mcpx/config.yml` location and `$(mcpx.home)` |
| `PATH` | Server commands are looked up here |

mcpx also passes through the entire current environment to server subprocesses, with any `env:` entries from config appended.

## File Paths

### Config files

| Path | Purpose |
|------|---------|
| `~/.mcpx/config.yml` | Global config (user-level) |
| `.mcpx/config.yml` | Project config (walk-up search from cwd) |

### Daemon files

| Path | Purpose |
|------|---------|
| `/tmp/mcpx-<server>-<uid>.sock` | Unix socket (mode 0600) |
| `/tmp/mcpx-<server>-<uid>.pid` | PID file |
| `/tmp/mcpx-<server>-<uid>.log` | Daemon log |

### Keychain

| OS | Backend |
|----|---------|
| macOS | Keychain (service: `mcpx`) |
| Linux | Secret Service API (GNOME Keyring / KWallet) |
| Windows | Windows Credential Manager |

## Color Behavior

mcpx uses [fatih/color](https://github.com/fatih/color) which automatically disables color when:

- `NO_COLOR` is set (any value)
- stdout is not a terminal (piped or redirected)
- `TERM=dumb`

You can also use `--json` or `--quiet` to get uncolored output.
