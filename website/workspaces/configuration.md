# Workspace Configuration

## Schema

```yaml
servers:
  <server-name>:
    workspaces:
      - name: string              # workspace identifier (required)
        path: string              # path relative to project root (required)
        lifecycle:                # workspace-specific lifecycle hooks
          on_connect:
            - tool: string
              args: { key: value }
        security:                 # workspace-specific security
          mode: string            # read-only, editing, custom
          allowed_tools: [string]
          blocked_tools: [string]
          policies: [...]
```

## Fields

### `name` (required)

A unique identifier for the workspace. Used in error messages and logging.

```yaml
name: frontend
name: api-service
name: shared-lib
```

### `path` (required)

Path relative to the project root (the directory containing `.mcpx/config.yml`). mcpx checks if the current working directory is inside this path.

```yaml
path: packages/web
path: services/api
path: libs/shared-rust
```

### `lifecycle`

Overrides the server-level lifecycle hooks when this workspace is active. Most commonly used to activate a specific project in Serena:

```yaml
lifecycle:
  on_connect:
    - tool: activate_project
      args:
        project: "$(mcpx.project_root)/packages/web"
```

Hook arguments support [dynamic variables](/guide/variables).

### `security`

Overrides the server-level security config when this workspace is active. Supports all the same fields as server-level security:

```yaml
security:
  mode: read-only
  policies:
    - name: restrict
      match:
        args:
          relative_path: { allow_prefix: ["src/"] }
      action: deny
      message: "Writes restricted to src/"
```

## Complete Example

A monorepo with three workspaces, each with different security profiles:

```yaml
servers:
  serena:
    command: serena
    args: [start-mcp-server, --context=claude-code]
    daemon: true
    startup_timeout: 30s

    # Defaults (when not in any workspace)
    lifecycle:
      on_connect:
        - tool: activate_project
          args: { project: "$(mcpx.project_root)" }
    security:
      mode: editing

    workspaces:
      # TypeScript frontend — full editing, restricted to src/
      - name: web
        path: packages/web
        lifecycle:
          on_connect:
            - tool: activate_project
              args: { project: "$(mcpx.project_root)/packages/web" }
        security:
          mode: editing
          policies:
            - name: web-src-only
              match:
                tools: ["replace_*", "insert_*", "rename_*"]
                args:
                  relative_path: { allow_prefix: ["src/", "public/", "./"] }
              action: deny
              message: "Web writes restricted to src/ and public/"

      # Go backend — full editing, restricted to cmd/internal/
      - name: api
        path: services/api
        lifecycle:
          on_connect:
            - tool: activate_project
              args: { project: "$(mcpx.project_root)/services/api" }
        security:
          mode: editing
          policies:
            - name: api-src-only
              match:
                tools: ["replace_*", "insert_*", "rename_*"]
                args:
                  relative_path: { allow_prefix: ["cmd/", "internal/", "pkg/", "./"] }
              action: deny
              message: "API writes restricted to cmd/, internal/, pkg/"

      # ML pipeline — read-only (no modifications)
      - name: ml-pipeline
        path: services/ml-pipeline
        lifecycle:
          on_connect:
            - tool: activate_project
              args: { project: "$(mcpx.project_root)/services/ml-pipeline" }
        security:
          mode: read-only
```

## Validation

mcpx validates workspace config at startup:

- `name` is required and must not be empty
- `path` is required and must not be empty
- Lifecycle hooks must have a `tool` name
- Security policies must have valid actions (`allow`, `deny`, `warn`)

```bash
$ mcpx list
error: config: server "serena": workspace "" missing name
```
