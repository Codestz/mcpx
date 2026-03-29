# Workspaces in Practice

Workspaces aren't just for monorepos. Any directory tree with multiple projects can use them — a monorepo, a folder of unrelated projects, or even nested monorepos inside a project folder.

The only requirement: place `.mcpx/config.yml` at the root of your directory tree and list each project as a workspace.

## Three Patterns

### Pattern 1: Monorepo

A single repository with multiple packages:

```
acme-platform/
├── .mcpx/config.yml
├── packages/web/          ← workspace
├── services/api/          ← workspace
└── services/ml-pipeline/  ← workspace
```

### Pattern 2: Multi-project folder

Unrelated projects in the same directory, each with its own `.serena`:

```
~/Projects/
├── .mcpx/config.yml
├── my-go-api/             ← workspace
├── react-dashboard/       ← workspace
├── ml-experiments/        ← workspace
└── rust-cli-tool/         ← workspace
```

Same config, same daemon, same security — just different project paths.

### Pattern 3: Nested monorepos

A project folder that contains standalone projects *and* a monorepo with its own inner workspaces:

```
~/Projects/
├── .mcpx/config.yml
├── my-go-api/                    ← workspace
├── react-dashboard/              ← workspace
└── acme-platform/                ← workspace (outer)
    ├── packages/web/             ← workspace (inner)
    ├── services/api/             ← workspace (inner)
    └── services/ml-pipeline/     ← workspace (inner)
```

This works because mcpx uses **longest path match**. When you `cd` into `acme-platform/services/api/`, both `acme-platform` and `acme-platform/services/api` match — but the inner one is more specific, so it wins.

If you `cd` into `acme-platform/` root (not inside any inner workspace), it falls back to the `acme-platform` outer workspace.

## The Config

Here's how all three patterns look in one config file:

```yaml
# ~/Projects/.mcpx/config.yml

security:
  enabled: true
  global:
    audit:
      enabled: true
      log: "$(mcpx.project_root)/.mcpx/audit.jsonl"
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

    # Default — used when not in any workspace
    lifecycle:
      on_connect:
        - tool: activate_project
          args: { project: "$(mcpx.project_root)" }
    security:
      mode: editing

    workspaces:
      # ── Standalone projects ──
      - name: my-go-api
        path: my-go-api
        lifecycle:
          on_connect:
            - tool: activate_project
              args: { project: "$(mcpx.project_root)/my-go-api" }
        security:
          mode: editing

      - name: react-dashboard
        path: react-dashboard
        lifecycle:
          on_connect:
            - tool: activate_project
              args: { project: "$(mcpx.project_root)/react-dashboard" }
        security:
          mode: editing

      - name: ml-experiments
        path: ml-experiments
        lifecycle:
          on_connect:
            - tool: activate_project
              args: { project: "$(mcpx.project_root)/ml-experiments" }
        security:
          mode: read-only

      # ── Monorepo (outer) ──
      - name: acme-platform
        path: acme-platform
        lifecycle:
          on_connect:
            - tool: activate_project
              args: { project: "$(mcpx.project_root)/acme-platform" }
        security:
          mode: editing

      # ── Monorepo inner workspaces ──
      - name: acme-web
        path: acme-platform/packages/web
        lifecycle:
          on_connect:
            - tool: activate_project
              args: { project: "$(mcpx.project_root)/acme-platform/packages/web" }
        security:
          mode: editing
          policies:
            - name: web-src-only
              match:
                tools: ["replace_*", "insert_*", "rename_*"]
                args:
                  relative_path:
                    allow_prefix: ["src/", "public/", "./"]
              action: deny
              message: "Web: writes restricted to src/ and public/"

      - name: acme-api
        path: acme-platform/services/api
        lifecycle:
          on_connect:
            - tool: activate_project
              args: { project: "$(mcpx.project_root)/acme-platform/services/api" }
        security:
          mode: editing
          policies:
            - name: api-src-only
              match:
                tools: ["replace_*", "insert_*", "rename_*"]
                args:
                  relative_path:
                    allow_prefix: ["cmd/", "internal/", "pkg/", "./"]
              action: deny
              message: "API: writes restricted to cmd/, internal/, pkg/"

      - name: acme-ml
        path: acme-platform/services/ml-pipeline
        lifecycle:
          on_connect:
            - tool: activate_project
              args: { project: "$(mcpx.project_root)/acme-platform/services/ml-pipeline" }
        security:
          mode: read-only
```

## How Detection Works

1. mcpx walks up from `cwd` to find `.mcpx/config.yml` (the project root)
2. For each workspace, it checks: is `cwd` inside `<project_root>/<workspace_path>`?
3. **Longest path wins** — if multiple workspaces match, the most specific one is selected
4. If no workspace matches, server-level defaults apply

```bash
# You're here:
$ cd ~/Projects/acme-platform/services/api/internal/handlers

# Matches:
#   acme-platform              (path length: 14)
#   acme-platform/services/api (path length: 28) ← wins (longest)

# Result: acme-api workspace activated with API security policies
```

## Testing It

```bash
# Standalone project
$ cd ~/Projects/my-go-api
$ mcpx serena find_symbol --name "Handler"
# → activates my-go-api project

# Inner monorepo workspace
$ cd ~/Projects/acme-platform/services/api
$ mcpx serena find_symbol --name "UserHandler"
# → activates acme-api workspace (not acme-platform outer)

# Monorepo root (no inner workspace match)
$ cd ~/Projects/acme-platform
$ mcpx serena list_dir --relative_path "." --recursive false
# → activates acme-platform outer workspace

# Read-only enforcement on ML projects
$ cd ~/Projects/ml-experiments
$ mcpx serena replace_symbol_body --name "Model" --body "..." --relative_path "model.py"
# error: read-only mode denied tool "replace_symbol_body"

# Path traversal blocked everywhere
$ cd ~/Projects/my-go-api
$ mcpx serena find_symbol --name "x" --relative_path "../../../etc/passwd"
# error: policy "no-path-traversal" denied tool "find_symbol"
```

## Key Takeaways

- **Workspaces are just paths** — they don't need to be inside a monorepo, a git repo, or any special structure
- **One daemon, many projects** — Serena runs once, mcpx routes to the right project based on `cwd`
- **Nesting works** — inner workspaces override outer ones via longest-path match
- **Security is per-workspace** — read-only for ML, path-restricted for API, full editing for frontend
- **Audit covers everything** — every call across every workspace goes through the same audit log
