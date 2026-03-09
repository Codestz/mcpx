# Task Completion Checklist

After completing any code change, run these in order:

1. go vet ./...           - must pass with no issues
2. go test -race ./...    - all tests must pass, no race conditions
3. go build ./cmd/mcpx    - must compile successfully

## When adding new features
- Add table-driven tests with subtests
- Wrap errors with package context
- Update CLI help text if adding commands/flags
- Keep packages acyclic: config and secret are leaves, cli is the only orchestrator

## When modifying existing code
- Run the specific package tests first: go test -race ./internal/<pkg>/
- Check for broken references if renaming symbols
- Ensure backward compatibility of config format