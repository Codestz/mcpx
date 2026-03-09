# Shell Completion

mcpx provides shell completions with dynamic server and tool name suggestions.

## Setup

### Bash

```bash
# Add to ~/.bashrc
eval "$(mcpx completion bash)"
```

### Zsh

```bash
# Add to ~/.zshrc
eval "$(mcpx completion zsh)"
```

### Fish

```bash
mcpx completion fish | source

# Or persist:
mcpx completion fish > ~/.config/fish/completions/mcpx.fish
```

### PowerShell

```powershell
mcpx completion powershell | Out-String | Invoke-Expression
```

## What Gets Completed

| Context | Completions |
|---------|-------------|
| `mcpx <TAB>` | Server names, static commands (list, ping, etc.) |
| `mcpx ping <TAB>` | Server names from config |
| `mcpx list <TAB>` | Server names from config |
| `mcpx serena <TAB>` | Tool names (connects to server to discover) |
| `mcpx daemon stop <TAB>` | Server names from config |

### Dynamic tool completion

When you tab-complete after a server name (e.g., `mcpx serena <TAB>`), mcpx connects to the server to discover available tools. This requires the server to be reachable — for daemon servers, the daemon must be running.

If the server is not available, completion silently returns no suggestions.

## Verify

After sourcing completions, test:

```bash
mcpx <TAB><TAB>
# Should show: completion  configure  daemon  init  list  ping  secret  serena  version
```
