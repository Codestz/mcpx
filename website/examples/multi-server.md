# Example: Multi-Server Project

A project using multiple MCP servers to give AI agents comprehensive tooling.

## Scenario

A web application where the AI needs:
- **Serena** — semantic code navigation
- **Sequential Thinking** — structured reasoning for complex decisions
- **Filesystem** — file operations in specific directories

## Config

### Project config (`.mcpx/config.yml`)

```yaml
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
    daemon: true
    startup_timeout: "120s"
```

### Global config (`~/.mcpx/config.yml`)

```yaml
servers:
  sequential-thinking:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-sequential-thinking"]

  filesystem:
    command: npx
    args:
      - -y
      - "@modelcontextprotocol/server-filesystem"
      - "$(env.HOME)/projects"
```

### Result

```bash
mcpx list
#   serena                uvx (daemon)     ← from project config
#   sequential-thinking   npx              ← from global config
#   filesystem            npx              ← from global config
```

Three servers, zero tokens of context overhead.

## Usage Patterns

### Code exploration

```bash
# Find a class
mcpx serena find_symbol --name_path_pattern "PaymentService" --include_body true

# Find all references
mcpx serena find_referencing_symbols --name_path "processPayment" \
  --relative_path "src/payments/service.go"
```

### Structured thinking

```bash
# Complex architectural decision
mcpx sequential-thinking think \
  --problem "Should we split PaymentService into separate microservices?" \
  --total_thoughts 5
```

### Composing servers

```bash
# Find a symbol, then think about it
mcpx serena find_symbol --name_path_pattern "PaymentService" --include_body true --json \
  | mcpx sequential-thinking think --problem - --total_thoughts 3
```

The output of one server becomes the input to another — standard UNIX piping.

## AI Agent Setup

```markdown
## Tools (CLAUDE.md)

This project uses mcpx for tool access:
- `mcpx list` — discover servers
- `mcpx <server> <tool> --help` — tool details

Servers:
- serena — semantic code navigation (daemon, fast after first call)
- sequential-thinking — structured reasoning
- filesystem — file operations in ~/projects
```

## Token Cost Comparison

| Approach | Context Cost |
|----------|-------------|
| 3 MCP servers loaded natively | ~33K tokens |
| Same 3 servers via mcpx | 0 tokens (tools called on demand) |
| 10 MCP servers loaded natively | ~100K+ tokens |
| Same 10 servers via mcpx | Still 0 tokens |

As the number of servers grows, the advantage compounds.
