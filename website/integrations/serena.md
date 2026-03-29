# Serena Integration

[Serena](https://github.com/oraios/serena) is a semantic code intelligence MCP server powered by Language Server Protocol (LSP). It provides deep code understanding — symbol search, references, refactoring — across dozens of languages.

mcpx + Serena is the recommended setup for AI-assisted development with full security and monorepo support.

## Why Serena + mcpx

| Challenge | Solution |
|-----------|----------|
| Serena exposes 30+ tools (~20K tokens) | mcpx calls tools on demand — 0 tokens upfront |
| Slow startup (LSP initialization) | Daemon mode keeps Serena warm between calls |
| Project activation required | Lifecycle hooks activate automatically |
| Monorepo with multiple languages | Workspace detection routes to the right project |
| Need to restrict write access | Security modes and path policies |

## Basic Config

```yaml
# .mcpx/config.yml
servers:
  serena:
    command: serena
    args:
      - start-mcp-server
      - --context=claude-code
    transport: stdio
    daemon: true
    startup_timeout: 30s
    lifecycle:
      on_connect:
        - tool: activate_project
          args:
            project: "$(mcpx.project_root)"
```

## First Run

```bash
# Verify connection
mcpx ping serena
# serena: ok (21 tools, 2847ms)  ← first run: slow (LSP init)

# Second call — daemon is warm
mcpx ping serena
# serena: ok (21 tools, 12ms)    ← fast
```

## Common Operations

### Discover tools

```bash
mcpx list serena -v
```

### Search for symbols

```bash
mcpx serena find_symbol --name_path_pattern "UserAuth" --include_body true
```

### Get file overview

```bash
mcpx serena get_symbols_overview --relative_path "src/auth/service.go"
```

### Find references

```bash
mcpx serena find_referencing_symbols --name_path "handleLogin" --relative_path "src/auth/handler.go"
```

### Search for patterns

```bash
mcpx serena search_for_pattern --substring_pattern "TODO|FIXME" --restrict_search_to_code_files true
```

### List directory

```bash
mcpx serena list_dir --relative_path "src/" --recursive true
```

## Lifecycle Hooks

Serena requires project activation before tools work. mcpx handles this automatically via lifecycle hooks:

```yaml
lifecycle:
  on_connect:
    - tool: activate_project
      args:
        project: "$(mcpx.project_root)"
```

Every time mcpx connects to the Serena daemon, it calls `activate_project` with the resolved project path. The AI agent never needs to think about activation.

**Important:** Onboarding is NOT automatic. It's a heavy operation (indexing, language server setup) that should be run manually:

```bash
mcpx serena onboarding
```

### Error Handling

If the project doesn't exist or hasn't been onboarded:

```
server "serena": lifecycle hook "activate_project" failed
  Error: No project found at /Users/you/myproject
  Hint:  Run "mcpx serena onboarding" to set up the project first
```

## Security

### Read-only mode

For code review sessions where the agent should analyze but not modify:

```yaml
security:
  mode: read-only
```

### Path restrictions

Allow writes only to source directories:

```yaml
security:
  mode: editing
  policies:
    - name: source-only
      match:
        tools: ["replace_*", "insert_*", "rename_*"]
        args:
          relative_path:
            allow_prefix: ["src/", "internal/", "cmd/", "./"]
      action: deny
      message: "Writes restricted to source directories"
```

## Monorepo Workspaces

For monorepos, define workspaces so mcpx activates the right Serena project based on your current directory:

```yaml
workspaces:
  - name: frontend
    path: packages/web
    lifecycle:
      on_connect:
        - tool: activate_project
          args: { project: "$(mcpx.project_root)/packages/web" }
    security:
      mode: editing

  - name: backend
    path: services/api
    lifecycle:
      on_connect:
        - tool: activate_project
          args: { project: "$(mcpx.project_root)/services/api" }
    security:
      mode: editing
```

Each workspace needs its own `.serena/project.yml` with the language configured.

See [Serena Monorepo Walkthrough](/workspaces/serena-monorepo) for a complete 5-workspace example.

## AI Agent Integration

Add to your `CLAUDE.md`:

```markdown
## Code Navigation

Use mcpx for semantic code search:
- `mcpx serena find_symbol --name_path_pattern "ClassName" --include_body true`
- `mcpx serena get_symbols_overview --relative_path "path/to/file.go"`
- `mcpx serena find_referencing_symbols --name_path "funcName" --relative_path "file.go"`
- `mcpx serena search_for_pattern --substring_pattern "pattern"`

Run `mcpx serena <tool> --help` for full flag details.
```

## Generate Reference

Create a compact reference for your CLAUDE.md:

```bash
mcpx serena generate
```

This outputs all tools and flags in a format optimized for AI agent context.
