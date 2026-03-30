# Serena Integration

[Serena](https://github.com/oraios/serena) is a semantic code intelligence MCP server powered by Language Server Protocol (LSP). It provides deep code understanding — symbol search, references, refactoring — across dozens of languages.

mcpx + Serena is the recommended setup for AI-assisted development with full security.

## Why Serena + mcpx

| Challenge | Solution |
|-----------|----------|
| Serena exposes 30+ tools (~20K tokens) | mcpx calls tools on demand — 0 tokens upfront |
| Slow startup (LSP initialization) | Daemon mode keeps Serena warm between calls |
| Project activation required | Daemon mode with auto-activation |
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
