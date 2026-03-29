# Security Modes

Modes are one-line security presets that apply sensible defaults without writing individual policies.

## Available Modes

| Mode | Behavior |
|------|----------|
| `read-only` | Blocks all write/mutation tools |
| `editing` | Full access (no restrictions from mode) |
| `custom` | No mode-level restrictions, policies only |

## Configuration

```yaml
servers:
  serena:
    security:
      mode: read-only
```

## read-only

Denies any tool matching these patterns:

- `replace_*` — symbol/content replacement
- `insert_*` — content insertion
- `delete_*` — deletion operations
- `remove_*` — removal operations
- `create_*` — creation operations
- `update_*` — update operations
- `rename_*` — rename operations
- `write_*` — write operations
- `execute` — raw execution
- `drop_*` — database drops
- `alter_*` — schema alterations

```bash
$ mcpx serena replace_symbol_body --name_path "Config" --body "..." --relative_path "config.go"

error: server "serena": read-only mode denied tool "replace_symbol_body"
  Hint: set security.mode to "editing" or "custom" to allow write tools
```

Read tools like `find_symbol`, `search_for_pattern`, `list_dir`, `get_symbols_overview` work normally.

### When to use

- **Code review sessions** — the agent should read and analyze, not modify
- **ML pipelines** — prevent accidental changes to models or training data
- **Production databases** — allow queries, block mutations
- **Auditing** — inspect a codebase without risk of modification

## editing

Full access. The mode itself imposes no restrictions. Policies defined under `security.policies` still apply.

```yaml
servers:
  serena:
    security:
      mode: editing
      policies:
        - name: restrict-paths
          match:
            args:
              relative_path: { allow_prefix: ["src/", "internal/"] }
          action: deny
          message: "Writes restricted to source directories"
```

### When to use

- **Active development** — the agent can read and write code
- **Combined with policies** — mode allows writes, policies restrict where

## custom

No mode-level restrictions. Only policies are evaluated. This is the default when `mode` is not specified.

### When to use

- **Fine-grained control** — you want to define exactly which tools and arguments are allowed
- **Complex setups** — different tools need different rules

## Mode + Policy Interaction

Modes and policies work together. The mode is checked first, then policies:

```yaml
servers:
  serena:
    security:
      mode: editing                    # mode allows everything
      policies:
        - name: no-vendor             # policy restricts where
          match:
            args:
              relative_path: { deny_prefix: ["vendor/", "node_modules/"] }
          action: deny
          message: "Cannot modify vendor directories"
```

## Per-Workspace Modes

In a monorepo, each workspace can have its own mode:

```yaml
servers:
  serena:
    workspaces:
      - name: ml-pipeline
        path: services/ml
        security:
          mode: read-only        # ML code is read-only

      - name: api
        path: services/api
        security:
          mode: editing          # API code is editable
```
