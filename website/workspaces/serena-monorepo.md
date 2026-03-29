# Serena Monorepo Walkthrough

This guide walks through a real monorepo setup with 5 workspaces across 5 languages, demonstrating workspace auto-detection, per-workspace security, and lifecycle hooks.

## The Monorepo

```
acme-platform/
├── .mcpx/config.yml              ← mcpx config (one file for everything)
├── packages/
│   ├── web/                      ← React/TypeScript frontend
│   │   ├── .serena/project.yml
│   │   └── src/App.tsx
│   └── mobile/                   ← React Native app
│       ├── .serena/project.yml
│       └── src/HomeScreen.tsx
├── services/
│   ├── api/                      ← Go backend
│   │   ├── .serena/project.yml
│   │   ├── cmd/main.go
│   │   └── internal/handlers/
│   └── ml-pipeline/              ← Python ML pipeline
│       ├── .serena/project.yml
│       └── src/pipeline.py
└── libs/
    └── shared-rust/              ← Rust shared library
        ├── .serena/project.yml
        └── src/lib.rs
```

Each workspace has its own `.serena/project.yml` with the language configured. mcpx handles activating the right project automatically.

## The Config

```yaml
# acme-platform/.mcpx/config.yml

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
                  relative_path:
                    allow_prefix: ["src/", "public/", "./"]
              action: deny
              message: "Web: writes restricted to src/ and public/"

      - name: mobile
        path: packages/mobile
        lifecycle:
          on_connect:
            - tool: activate_project
              args: { project: "$(mcpx.project_root)/packages/mobile" }
        security:
          mode: editing

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
                  relative_path:
                    allow_prefix: ["cmd/", "internal/", "pkg/", "./"]
              action: deny
              message: "API: writes restricted to cmd/, internal/, pkg/"

      - name: ml-pipeline
        path: services/ml-pipeline
        lifecycle:
          on_connect:
            - tool: activate_project
              args: { project: "$(mcpx.project_root)/services/ml-pipeline" }
        security:
          mode: read-only

      - name: shared-rust
        path: libs/shared-rust
        lifecycle:
          on_connect:
            - tool: activate_project
              args: { project: "$(mcpx.project_root)/libs/shared-rust" }
        security:
          mode: editing
          policies:
            - name: rust-src-only
              match:
                tools: ["replace_*", "insert_*", "rename_*"]
                args:
                  relative_path:
                    allow_prefix: ["src/", "./"]
              action: deny
              message: "Rust lib: writes restricted to src/"
```

## Testing It

### Go workspace — find a symbol

```bash
$ cd services/api
$ mcpx serena find_symbol --name_path_pattern "UserHandler" --relative_path "internal/handlers/users.go"

[{"name_path": "UserHandler", "kind": "Struct", "relative_path": "internal/handlers/users.go", ...}]
```

Serena activated `acme-api` (Go project) and found the struct.

### TypeScript workspace — find a component

```bash
$ cd packages/web
$ mcpx serena find_symbol --name_path_pattern "Dashboard" --relative_path "src/App.tsx"

[{"name_path": "Dashboard", "kind": "Constant", "relative_path": "src/App.tsx", ...}]
```

Serena switched to `acme-web` (TypeScript project) and found the React component.

### Python workspace — search for a class

```bash
$ cd services/ml-pipeline
$ mcpx serena search_for_pattern --substring_pattern "class ChurnPredictor" --relative_path "src"

{"src/pipeline.py": ["  >  26:class ChurnPredictor:"]}
```

### Read-only enforcement on ML pipeline

```bash
$ cd services/ml-pipeline
$ mcpx serena replace_symbol_body --name_path "ChurnPredictor" --relative_path "src/pipeline.py" --body "..."

error: server "serena": policy "(read-only mode)" denied tool "replace_symbol_body"
  Reason: server "serena": read-only mode denied tool "replace_symbol_body"
  Hint: set security.mode to "editing" or "custom" to allow write tools
```

### Path restriction on API

```bash
$ cd services/api
$ mcpx serena replace_symbol_body --name_path "Handler" --relative_path "vendor/hack.go" --body "..."

error: server "serena": policy "api-src-only" denied tool "replace_symbol_body"
  Reason: API: writes restricted to cmd/, internal/, pkg/
  relative_path = "vendor/hack.go"
```

### Global path traversal protection

```bash
$ cd packages/web
$ mcpx serena find_symbol --name_path_pattern "Config" --relative_path "../../../etc/passwd"

error: server "serena": policy "no-path-traversal" denied tool "find_symbol"
  Reason: Path traversal blocked
  relative_path = "../../../etc/passwd"
```

## Audit Trail

Every call is recorded:

```jsonl
{"timestamp":"2026-03-28T18:55:52Z","server":"serena","tool":"find_symbol","args":{...},"action":"allowed"}
{"timestamp":"2026-03-28T18:56:34Z","server":"serena","tool":"replace_symbol_body","args":{...},"action":"denied","policy_name":"(read-only mode)"}
{"timestamp":"2026-03-28T18:56:39Z","server":"serena","tool":"replace_symbol_body","args":{...},"action":"denied","policy_name":"api-src-only"}
{"timestamp":"2026-03-28T18:56:44Z","server":"serena","tool":"find_symbol","args":{...},"action":"denied","policy_name":"no-path-traversal"}
```

## Summary

| Workspace | Language | Security | Writes Allowed |
|-----------|----------|----------|---------------|
| web | TypeScript | editing + path policy | `src/`, `public/` |
| mobile | TypeScript | editing | everywhere |
| api | Go | editing + path policy | `cmd/`, `internal/`, `pkg/` |
| ml-pipeline | Python | read-only | nowhere |
| shared-rust | Rust | editing + path policy | `src/` |

One config file. Five languages. Five security profiles. Zero manual switching.
