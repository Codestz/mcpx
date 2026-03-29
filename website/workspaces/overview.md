# Workspaces Overview

Workspaces solve the monorepo problem: a single repository with multiple projects, each needing its own MCP server configuration, lifecycle hooks, and security rules.

## The Problem

In a monorepo like this:

```
acme-platform/
├── packages/web/          ← React/TypeScript
├── packages/mobile/       ← React Native
├── services/api/          ← Go backend
├── services/ml-pipeline/  ← Python ML
└── libs/shared-rust/      ← Rust library
```

Each directory is a different project with its own language, its own Serena configuration, and its own security needs. Without workspaces, you'd need to manually switch project contexts every time you `cd` into a different directory.

## The Solution

mcpx auto-detects which workspace you're in based on your current working directory and applies the right lifecycle hooks and security rules:

```yaml
# .mcpx/config.yml (at monorepo root)
servers:
  serena:
    command: serena
    daemon: true
    workspaces:
      - name: web
        path: packages/web
        lifecycle:
          on_connect:
            - tool: activate_project
              args: { project: "$(mcpx.project_root)/packages/web" }

      - name: api
        path: services/api
        lifecycle:
          on_connect:
            - tool: activate_project
              args: { project: "$(mcpx.project_root)/services/api" }
```

```bash
$ cd packages/web
$ mcpx serena find_symbol --name "Dashboard"
# Automatically activates web workspace, finds TypeScript component

$ cd services/api
$ mcpx serena find_symbol --name "Handler"
# Automatically activates API workspace, finds Go struct
```

## How Detection Works

1. mcpx walks up from `cwd` to find `.mcpx/config.yml` (the project root)
2. For each configured workspace, it checks if `cwd` is inside `<project_root>/<workspace_path>`
3. The most specific match wins (longest path)
4. If no workspace matches, server-level defaults apply

## What Workspaces Control

Each workspace can override:

- **Lifecycle hooks** — which project to activate, which setup to run
- **Security policies** — which tools are allowed, which paths are writable

```yaml
workspaces:
  - name: ml-pipeline
    path: services/ml
    lifecycle:
      on_connect:
        - tool: activate_project
          args: { project: "$(mcpx.project_root)/services/ml" }
    security:
      mode: read-only   # prevent accidental model changes

  - name: api
    path: services/api
    lifecycle:
      on_connect:
        - tool: activate_project
          args: { project: "$(mcpx.project_root)/services/api" }
    security:
      mode: editing
      policies:
        - name: api-paths
          match:
            tools: ["replace_*", "insert_*"]
            args:
              relative_path: { allow_prefix: ["cmd/", "internal/"] }
          action: deny
          message: "API writes restricted to cmd/ and internal/"
```

## Fallback Behavior

When `cwd` is not inside any workspace (e.g. at the monorepo root), the server-level `lifecycle` and `security` settings apply:

```yaml
servers:
  serena:
    # Default — used when not in a workspace
    lifecycle:
      on_connect:
        - tool: activate_project
          args: { project: "$(mcpx.project_root)" }
    security:
      mode: editing

    # Workspace overrides
    workspaces:
      - name: web
        path: packages/web
        # ...
```

## Next Steps

- [Workspace Configuration](/workspaces/configuration) — full YAML reference
- [Serena Monorepo](/workspaces/serena-monorepo) — complete walkthrough with 5 workspaces
