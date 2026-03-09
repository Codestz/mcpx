# Config Schema

Complete YAML schema reference for `.mcpx/config.yml`.

## Full Schema

```yaml
servers:
  <server-name>:
    # Required
    command: string

    # Optional
    args: [string]
    transport: "stdio"         # default: "stdio"
    daemon: bool               # default: false
    startup_timeout: string    # default: "30s"
    env:
      <KEY>: <value>
```

## Field Reference

### `servers`

Top-level map of server names to their configurations. Server names become CLI subcommands.

**Naming rules:**
- Use lowercase letters, digits, and hyphens
- Avoid names that conflict with built-in commands (`list`, `ping`, `daemon`, `secret`, `init`, `version`, `completion`, `configure`)

### `command` (required)

The executable to spawn. Must be in `$PATH` or an absolute path.

```yaml
command: uvx
command: npx
command: /usr/local/bin/my-server
```

### `args`

List of arguments passed to the command. Supports [dynamic variables](/guide/variables).

```yaml
args:
  - --from
  - git+https://github.com/oraios/serena
  - serena
  - start-mcp-server
  - --project
  - "$(mcpx.project_root)"
```

### `transport`

Communication protocol. Currently only `stdio` is supported.

```yaml
transport: stdio    # default
```

### `daemon`

When `true`, the server is spawned once as a background process and kept alive between calls. Connections are made via unix socket.

```yaml
daemon: true
```

See [Daemon Mode](/guide/daemon-mode).

### `startup_timeout`

Maximum time to wait for the server to become responsive. Accepts Go duration strings.

```yaml
startup_timeout: "60s"     # 60 seconds
startup_timeout: "2m"      # 2 minutes
startup_timeout: "30s"     # default
```

### `env`

Extra environment variables injected into the server process. Supports [dynamic variables](/guide/variables).

```yaml
env:
  GITHUB_TOKEN: "$(secret.github_token)"
  NODE_ENV: production
  DEBUG: "true"
```

These are appended to the current environment — they don't replace it.

## Example: Complete Config

```yaml
servers:
  serena:
    command: uvx
    args:
      - --from
      - git+https://github.com/oraios/serena
      - serena
      - start-mcp-server
      - --project
      - "$(mcpx.project_root)"
    transport: stdio
    daemon: true
    startup_timeout: "60s"

  sequential-thinking:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-sequential-thinking"]

  filesystem:
    command: npx
    args:
      - -y
      - "@modelcontextprotocol/server-filesystem"
      - "$(env.HOME)/projects"
    env:
      NODE_ENV: production
```

## Config Location Search

mcpx searches for project config by walking up directories from the current working directory:

```
/home/user/project/src/pkg/    ← cwd
/home/user/project/src/
/home/user/project/            ← .mcpx/config.yml found here
```

The directory containing `.mcpx/` becomes the **project root**, available as `$(mcpx.project_root)`.
