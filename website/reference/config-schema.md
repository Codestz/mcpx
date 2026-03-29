# Config Schema

Complete YAML schema reference for `.mcpx/config.yml`.

## Full Schema

```yaml
# Top-level security (applies to all servers)
security:
  enabled: bool
  global:
    audit:
      enabled: bool
      log: string                # path, supports $(variables)
      redact: [string]
    rate_limit:
      max_calls_per_minute: int
      max_calls_per_tool: int
    policies:
      - name: string
        match:
          tools: [string]        # glob patterns
          args:
            <name-pattern>:
              deny_pattern: regex
              allow_prefix: [string]
              deny_prefix: [string]
          content:
            target: string       # e.g. "args.sql"
            deny_pattern: regex
            require_pattern: regex
            when: regex
        action: string           # allow, deny, warn
        message: string

# Server definitions
servers:
  <server-name>:
    # Connection
    command: string              # required for stdio
    args: [string]
    transport: string            # stdio (default), http, sse
    url: string                  # required for http/sse
    headers: { key: value }
    auth:
      type: string               # e.g. "bearer"
      token: string
    daemon: bool                 # default: false
    startup_timeout: string      # default: "30s"
    env: { KEY: value }

    # Security (per-server)
    security:
      mode: string               # read-only, editing, custom
      allowed_tools: [string]
      blocked_tools: [string]
      policies: [...]            # same as global policies

    # Lifecycle
    lifecycle:
      on_connect:
        - tool: string
          args: { key: value }

    # Workspaces (monorepo)
    workspaces:
      - name: string
        path: string             # relative to project root
        lifecycle: { ... }       # overrides server lifecycle
        security: { ... }        # overrides server security
```

## Field Reference

### Top-Level

#### `security`

Global security configuration. See [Security Overview](/security/overview).

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Enable security policy evaluation |
| `global.audit.enabled` | bool | `false` | Enable audit logging |
| `global.audit.log` | string | — | Path to JSONL log file |
| `global.audit.redact` | [string] | `[]` | Patterns for value redaction |
| `global.rate_limit.max_calls_per_minute` | int | — | Rate limit across all tools |
| `global.policies` | [Policy] | `[]` | Global security policies |

### Server Fields

#### `command` (required for stdio)

The executable to spawn. Must be in `$PATH` or an absolute path.

```yaml
command: serena
command: npx
command: /usr/local/bin/my-server
```

#### `args`

List of arguments passed to the command. Supports [dynamic variables](/guide/variables).

```yaml
args:
  - start-mcp-server
  - --context=claude-code
  - --project
  - "$(mcpx.project_root)"
```

#### `transport`

Communication protocol.

| Value | Description |
|-------|-------------|
| `stdio` | Subprocess stdin/stdout (default) |
| `http` | Streamable HTTP (MCP 2025-11-25 spec) |
| `sse` | Server-Sent Events |

#### `url` (required for http/sse)

Server URL for remote transports.

```yaml
url: "http://localhost:8080/mcp"
url: "$(secret.mcp_url)"
```

#### `headers`

HTTP headers for remote transports. Supports [dynamic variables](/guide/variables).

```yaml
headers:
  X-Api-Key: "$(secret.api_key)"
```

#### `auth`

Authentication configuration for remote transports.

```yaml
auth:
  type: bearer
  token: "$(secret.auth_token)"
```

#### `daemon`

When `true`, the server is spawned once and kept alive between calls via unix socket. See [Daemon Mode](/guide/daemon-mode).

#### `startup_timeout`

Maximum time to wait for the server to become responsive. Accepts Go duration strings.

```yaml
startup_timeout: "60s"
startup_timeout: "2m"
```

#### `env`

Extra environment variables injected into the server process.

```yaml
env:
  GITHUB_TOKEN: "$(secret.github_token)"
  NODE_ENV: production
```

#### `security`

Per-server security configuration. See [Security Policies](/security/policies) and [Modes](/security/modes).

```yaml
security:
  mode: read-only
  allowed_tools: [find_*, search_*, list_*]
  blocked_tools: [delete_*]
  policies:
    - name: restrict-paths
      match:
        args:
          relative_path: { allow_prefix: ["src/"] }
      action: deny
      message: "Restricted to src/"
```

#### `lifecycle`

Hooks that run after connecting to the server. See [Lifecycle Hooks](#lifecycle-hooks).

```yaml
lifecycle:
  on_connect:
    - tool: activate_project
      args: { project: "$(mcpx.project_root)" }
```

#### `workspaces`

Monorepo workspace definitions. See [Workspaces](/workspaces/overview).

```yaml
workspaces:
  - name: frontend
    path: packages/web
    lifecycle: { ... }
    security: { ... }
```

## Lifecycle Hooks

Hooks run automatically after the MCP handshake completes. They execute sequentially — if any hook fails, remaining hooks are skipped and an error is returned.

```yaml
lifecycle:
  on_connect:
    - tool: activate_project
      args:
        project: "$(mcpx.project_root)"
```

Hook arguments support [dynamic variables](/guide/variables). The `tool` field is the MCP tool name to call on the server.

**Important:** Hooks should be lightweight and idempotent. Heavy operations like onboarding should be run manually.

## Example: Complete Config

```yaml
security:
  enabled: true
  global:
    audit:
      enabled: true
      log: "$(mcpx.project_root)/.mcpx/audit.jsonl"
      redact: ["$(secret.*)"]
    policies:
      - name: no-path-traversal
        match:
          args:
            "*path*":
              deny_pattern: "\\.\\.\\/|\\.\\.\\\\\\/"
        action: deny
        message: "Path traversal blocked"

servers:
  serena:
    command: serena
    args: [start-mcp-server, --context=claude-code]
    daemon: true
    startup_timeout: 30s
    lifecycle:
      on_connect:
        - tool: activate_project
          args: { project: "$(mcpx.project_root)" }
    security:
      mode: editing
    workspaces:
      - name: api
        path: services/api
        lifecycle:
          on_connect:
            - tool: activate_project
              args: { project: "$(mcpx.project_root)/services/api" }
        security:
          mode: editing

  postgres:
    command: postgres-mcp
    env:
      DATABASE_URL: "$(secret.pg_url)"
    security:
      mode: read-only

  jira:
    command: jira-mcp
    transport: http
    url: "$(secret.jira_url)"
    auth:
      type: bearer
      token: "$(secret.jira_token)"
    security:
      mode: read-only
      allowed_tools: [search_issues, get_issue, list_projects]
```

## Config Location Search

mcpx searches for project config by walking up directories from the current working directory:

```
/home/user/project/src/pkg/    ← cwd
/home/user/project/src/
/home/user/project/            ← .mcpx/config.yml found here
```

The directory containing `.mcpx/` becomes the **project root**, available as `$(mcpx.project_root)`.
