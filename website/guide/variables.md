# Dynamic Variables

Variables use the `$(namespace.key)` syntax and are resolved at runtime.

## Available Variables

### `$(mcpx.*)`

| Variable | Value |
|----------|-------|
| `$(mcpx.project_root)` | Directory containing `.mcpx/config.yml` |
| `$(mcpx.cwd)` | Current working directory |
| `$(mcpx.home)` | User home directory |

### `$(git.*)`

| Variable | Value |
|----------|-------|
| `$(git.root)` | Git repository root |
| `$(git.branch)` | Current branch name |
| `$(git.remote)` | Remote URL (origin) |
| `$(git.commit)` | Current commit SHA |

::: warning
Git variables require a git repository. They produce an error if used outside one.
:::

### `$(env.*)`

Any environment variable:

```yaml
args:
  - "$(env.HOME)/projects"
  - "$(env.API_BASE_URL)"
```

Produces an error if the variable is not set.

### `$(secret.*)`

Secrets stored in the OS keychain via `mcpx secret set`:

```yaml
env:
  GITHUB_TOKEN: "$(secret.github_token)"
  API_KEY: "$(secret.openai_key)"
```

See [Secrets](/guide/secrets) for managing stored secrets.

### `$(sys.*)`

| Variable | Value |
|----------|-------|
| `$(sys.os)` | `linux`, `darwin`, or `windows` |
| `$(sys.arch)` | `amd64`, `arm64`, etc. |

## Usage

Variables can appear in `args` and `env` values:

```yaml
servers:
  serena:
    command: uvx
    args:
      - serena
      - --project
      - "$(mcpx.project_root)"
    env:
      TOKEN: "$(secret.github_token)"
      HOME_DIR: "$(env.HOME)"
```

### Multiple variables in one string

```yaml
args:
  - "$(env.HOME)/.config/$(sys.os)/settings.json"
```

### Variable in env values

```yaml
env:
  CONFIG_PATH: "$(mcpx.project_root)/config"
  GIT_REF: "$(git.branch)"
```

## Security

- Variables use a strict regex: `\$\(([a-z]+)\.([a-zA-Z0-9_.]+)\)`
- Unknown namespaces produce errors, not passthrough
- Secrets are resolved from the OS keychain at runtime — never written to disk
- No shell expansion occurs — `$(...)` is mcpx syntax, not shell syntax

## Debugging

Use `--dry-run` to see resolved values:

```bash
mcpx serena search_symbol --name "test" --dry-run
```

```
Dry run — nothing will be executed

Server:  serena
Tool:    search_symbol
Command: uvx --from git+https://github.com/oraios/serena serena start-mcp-server --project /home/user/myproject
Env:
  GITHUB_TOKEN=ghp_abc...
Arguments:
  {"name": "test"}
```

::: danger
`--dry-run` will show resolved secret values. Be careful when sharing output.
:::
