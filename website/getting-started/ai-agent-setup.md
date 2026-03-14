# AI Agent Setup

mcpx is designed for AI agents. Here's how to integrate it with popular AI coding tools.

## Claude Code

Add to your project's `CLAUDE.md`:

```markdown
## Tools

This project uses mcpx for MCP tool access. Available commands:
- `mcpx list` — discover servers and tools
- `mcpx <server> <tool> --help` — get usage for any tool
- `mcpx <server> info` — check server capabilities
- `mcpx <server> prompt list` — list available prompts
- `mcpx <server> resource list` — list available resources
- Call tools via Bash as needed

Configured servers:
- serena — semantic code search and navigation
- sequential-thinking — structured reasoning
```

That's it. Claude Code will discover and call tools through Bash when it needs them.

### Before (native MCP)

```
# .mcp.json — every tool schema loaded into context
{
  "mcpServers": {
    "serena": { ... },           # ~20K tokens of schemas
    "sequential-thinking": { ... }, # ~5K tokens
    "filesystem": { ... }        # ~8K tokens
  }
}
# Total: ~33K tokens consumed before any work
```

### After (mcpx)

```
# CLAUDE.md — 3 lines of text
# Total: ~50 tokens. AI calls mcpx when needed.
```

## Cursor

Add to your project's `.cursorrules`:

```
This project uses mcpx for MCP tool access.
Run `mcpx list` to discover available servers and tools.
Run `mcpx <server> <tool> --help` to see tool flags.
Call tools via terminal: `mcpx <server> <tool> --flags`
```

## Other AI Agents

The pattern works with any AI agent that can execute shell commands:

1. Tell the agent that `mcpx` is available
2. Tell it to use `mcpx list` for discovery
3. Tell it to use `--help` for tool details
4. Let it call tools via Bash

The agent's existing knowledge of shell commands handles the rest.

## Tips for AI Agents

### Efficient discovery

```bash
# One-shot: see all tools with all flags
mcpx list serena -v

# Even better: generate a reference doc
mcpx serena generate
```

The `generate` command creates a compact reference of all tools and flags that fits in a CLAUDE.md file.

### JSON mode for structured data

```bash
mcpx serena search_symbol --name "Auth" --json
```

AI agents often work better with JSON output they can parse.

### Stdin for long arguments

```bash
printf '{"name": "very long argument..."}' | mcpx serena search_symbol --stdin
```

### Dry run for debugging

```bash
mcpx serena search_symbol --name "Auth" --dry-run
```

Shows what would execute without running it — useful for the AI to verify its command before executing.

## Project Setup Checklist

1. Install mcpx: `go install github.com/codestz/mcpx/cmd/mcpx@latest`
2. Import or create config: `mcpx init` or create `.mcpx/config.yml`
3. Verify: `mcpx ping <server>`
4. Add instructions to `CLAUDE.md` / `.cursorrules`
5. Remove MCP servers from native config (optional — they still work, just cost tokens)
