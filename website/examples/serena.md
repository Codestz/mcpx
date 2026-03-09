# Example: Serena

[Serena](https://github.com/oraios/serena) is a semantic code search MCP server powered by LSP. It's a great example of a server that benefits from mcpx + daemon mode.

## Why Serena + mcpx

Serena exposes 21 tools with complex schemas. Loading them natively costs ~20K tokens of context. With mcpx, the AI calls tools on demand — zero upfront cost.

Serena also has slow startup (LSP initialization, indexing). Daemon mode keeps it warm.

## Config

```yaml
# .mcpx/config.yml
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
    startup_timeout: "120s"
```

## First Run

```bash
# Import if you have .mcp.json
mcpx init

# Or create config manually (above)

# Verify
mcpx ping serena
# serena: ok (21 tools, 2847ms)  ← first run: slow (LSP init)

# Second call
mcpx ping serena
# serena: ok (21 tools, 12ms)    ← daemon warm: fast
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
