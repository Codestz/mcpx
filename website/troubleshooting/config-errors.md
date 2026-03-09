# Config Errors

Exit code 2. Problems with configuration files or variable resolution.

## YAML Parse Error

```
error: config: load /home/user/.mcpx/config.yml: yaml: line 5: did not find expected key
```

**Cause:** Invalid YAML syntax.

**Common mistakes:**

```yaml
# Wrong: tabs instead of spaces
servers:
	serena:                    # YAML requires spaces, not tabs

# Wrong: missing quotes around special characters
args:
  - --project=$(mcpx.project_root)    # needs quotes

# Right
args:
  - "--project=$(mcpx.project_root)"

# Wrong: colon in unquoted value
env:
  URL: http://localhost:3000    # colon needs quoting

# Right
env:
  URL: "http://localhost:3000"
```

**Fix:** Use a YAML validator or check indentation carefully. YAML uses 2-space indentation by convention.

## Server Not Found

```
error: server "serna" not found. Available: serena
```

**Cause:** Typo in server name.

**Fix:** Check `mcpx list` for exact server names. Names are case-sensitive.

## Variable Resolution Failed

### Unknown namespace

```
error: server "serena": resolve args: unknown variable namespace "invalid" in "$(invalid.var)"
```

**Cause:** Using a namespace that doesn't exist.

**Valid namespaces:** `mcpx`, `git`, `env`, `secret`, `sys`

### Missing environment variable

```
error: server "serena": resolve args: env variable "API_KEY" not set
```

**Cause:** `$(env.API_KEY)` referenced but `API_KEY` isn't in the environment.

**Fix:** Set the variable or use `$(secret.*)` for sensitive values:
```bash
export API_KEY=your_key
# or
mcpx secret set api_key your_key
```

### Git variable outside repo

```
error: server "serena": resolve args: git: not a git repository
```

**Cause:** Using `$(git.root)` or similar outside a git repository.

**Fix:** Ensure you're in a git repo, or use `$(mcpx.project_root)` instead.

## Config Not Found

```
No servers configured.
Add servers to .mcpx/config.yml or ~/.mcpx/config.yml
```

**Cause:** mcpx couldn't find any config file.

**Fix:**
- Run `mcpx init` to import from `.mcp.json`
- Create `.mcpx/config.yml` in your project
- Create `~/.mcpx/config.yml` for global servers

## Merge Confusion

**Symptom:** A server isn't using the config you expect.

**How merge works:**
- mcpx loads `~/.mcpx/config.yml` (global) and `.mcpx/config.yml` (project)
- If both define the same server name, the **project version wins entirely**
- There is no field-level merge — it's all-or-nothing per server

**Debug:**
```bash
mcpx list                    # shows which servers are active
mcpx serena --help           # shows the command being used
mcpx serena tool --dry-run   # shows resolved command, args, env
```

## Command Required

```
error: config: server "myserver": command is required
```

**Cause:** A server entry is missing the `command` field.

**Fix:**
```yaml
servers:
  myserver:
    command: my-server-binary    # required
    args: [...]
```
