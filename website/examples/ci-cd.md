# Example: CI/CD Pipelines

Using mcpx in GitHub Actions and other CI environments.

## GitHub Actions

### Basic setup

```yaml
# .github/workflows/mcp-check.yml
name: MCP Tools Check
on: [push]

jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Install mcpx
        run: go install github.com/codestz/mcpx/cmd/mcpx@latest

      - name: Verify servers
        run: mcpx list
```

### With server calls

```yaml
      - name: Install server dependencies
        run: |
          pip install uvx  # for Serena
          # or: npm install -g @modelcontextprotocol/server-sequential-thinking

      - name: Run code analysis
        run: |
          mcpx serena get_symbols_overview --relative_path "src/main.go" --json
```

## Secrets in CI

In CI, there's no OS keychain. Use environment variables instead of `$(secret.*)`:

```yaml
# Config for CI
servers:
  my-api:
    command: my-mcp-server
    env:
      API_KEY: "$(env.API_KEY)"        # not $(secret.api_key)
```

```yaml
# GitHub Actions
      - name: Call API tool
        env:
          API_KEY: ${{ secrets.API_KEY }}
        run: mcpx my-api some_tool --arg value
```

## Daemon Mode in CI

Daemon mode works in CI but provides less benefit since CI jobs are short-lived. For CI, direct (non-daemon) mode is simpler:

```yaml
# CI config — daemon disabled
servers:
  serena:
    command: uvx
    args: [...]
    daemon: false              # override project config for CI
```

If your CI job makes many calls to the same server, daemon mode can still help:

```yaml
      - name: Start daemon
        run: mcpx ping serena  # starts daemon

      - name: Run multiple checks
        run: |
          mcpx serena find_symbol --name_path_pattern "Auth" --json
          mcpx serena find_symbol --name_path_pattern "Payment" --json
          mcpx serena get_symbols_overview --relative_path "src/main.go"

      - name: Stop daemon
        if: always()
        run: mcpx daemon stop-all
```

## Output in CI

Use `--json` for machine-parseable output:

```yaml
      - name: Check for TODOs
        run: |
          result=$(mcpx serena search_for_pattern \
            --substring_pattern "TODO|FIXME" \
            --restrict_search_to_code_files true \
            --json)
          echo "$result" | jq '.content[].text'
```

Use `--quiet` when you only care about the exit code:

```yaml
      - name: Verify server health
        run: mcpx ping serena --quiet
```

## Docker

```dockerfile
FROM golang:1.24 AS builder
RUN go install github.com/codestz/mcpx/cmd/mcpx@latest

FROM ubuntu:24.04
COPY --from=builder /go/bin/mcpx /usr/local/bin/mcpx
# Install your MCP server dependencies here
```

## Monorepo Setup

For monorepos where different services need different MCP configs:

```
monorepo/
  .mcpx/config.yml           # shared servers
  services/
    auth/
      .mcpx/config.yml       # auth-specific servers
    payments/
      .mcpx/config.yml       # payment-specific servers
```

When running from `services/auth/`, mcpx finds the nearest `.mcpx/` directory and uses that config, with global config filling gaps.
