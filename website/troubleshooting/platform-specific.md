# Platform-Specific Issues

## macOS

### Keychain Prompts {#macos-keychain-prompts}

**Symptom:** macOS shows a dialog asking to allow `mcpx` to access the keychain every time you use secrets.

**Fix:** When the dialog appears, click **Always Allow** to permanently grant access.

If you accidentally clicked "Deny":
1. Open **Keychain Access** (Applications > Utilities)
2. Find the `mcpx` entries
3. Right-click > Get Info > Access Control
4. Add `mcpx` to the list of allowed applications

### Gatekeeper Blocks Binary

**Symptom:** "mcpx cannot be opened because it is from an unidentified developer"

**Fix:**
```bash
# Remove quarantine flag
xattr -d com.apple.quarantine /usr/local/bin/mcpx
```

Or: System Preferences > Security & Privacy > "Allow Anyway"

### Apple Silicon vs Intel

If you installed the wrong architecture binary:

```bash
# Check your architecture
uname -m
# arm64 = Apple Silicon, x86_64 = Intel

# Reinstall the correct version
go install github.com/codestz/mcpx/cmd/mcpx@latest
```

`go install` automatically builds for your platform.

## Linux

### Secret Service Required {#linux-secret-service}

**Symptom:** `mcpx secret set` fails with a D-Bus error.

**Cause:** mcpx uses the Secret Service API (GNOME Keyring or KWallet). A running instance is required.

**Fix for GNOME:**
```bash
# Install GNOME Keyring
sudo apt install gnome-keyring

# Start it (if not running)
eval $(gnome-keyring-daemon --start)
export GNOME_KEYRING_CONTROL
export SSH_AUTH_SOCK
```

**Fix for KDE:**
```bash
sudo apt install kwalletmanager
```

**Headless servers / CI:**
On systems without a desktop environment, there's no keyring. Alternatives:
- Use `$(env.*)` instead of `$(secret.*)` and set environment variables directly
- Use a `.env` file loaded by your shell (not committed to git)

### Unix Socket Path Length

**Symptom:** Daemon fails to start with "file name too long" error.

**Cause:** Unix socket paths have a 108-character limit on Linux.

**Workaround:** Use short server names. The socket path is:
```
/tmp/mcpx-<server>-<uid>.sock
```

Server names longer than ~80 characters may hit this limit.

### Permission Issues with /tmp

Some hardened Linux systems restrict `/tmp` access.

**Fix:** If daemon sockets can't be created in `/tmp`, check your system's `tmp` permissions or mount options.

## Windows (WSL)

### WSL Recommended

mcpx is designed for Unix systems. On Windows, use WSL (Windows Subsystem for Linux):

```bash
# Install WSL
wsl --install

# Inside WSL
go install github.com/codestz/mcpx/cmd/mcpx@latest
```

### Native Windows

mcpx builds for Windows (`make build-all` produces `mcpx-windows-amd64.exe`), but:
- Daemon mode uses unix sockets, which are not available on native Windows
- Secrets use Windows Credential Manager (should work)
- Direct (non-daemon) server calls should work

### WSL Socket Paths

If using WSL, ensure socket paths don't cross the Windows/Linux filesystem boundary. Keep everything inside the WSL filesystem (`/home/user/...`), not in `/mnt/c/...`.

## Docker / Containers

### No Keychain Available

Containers typically don't have a keychain. Use environment variables instead:

```yaml
env:
  API_KEY: "$(env.API_KEY)"    # instead of $(secret.api_key)
```

Pass the variable when running the container:
```bash
docker run -e API_KEY=value myimage mcpx serena tool
```

### Daemon Mode in Containers

Daemon mode works in containers (unix sockets are available), but the daemon dies when the container stops. This is fine — it just restarts on the next call.
