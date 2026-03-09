# Contributing to mcpx

Thanks for your interest in contributing to mcpx! Here's how to get started.

## Development Setup

```bash
# Clone
git clone https://github.com/codestz/mcpx.git
cd mcpx

# Build
make build

# Run tests
make test

# Lint
make lint
```

**Requirements:** Go 1.24+

## Making Changes

1. **Fork** the repository
2. **Create a branch** from `main`: `git checkout -b my-feature`
3. **Make your changes** — keep them focused and minimal
4. **Add tests** for new functionality
5. **Run the checks:**
   ```bash
   go build ./cmd/mcpx
   go vet ./...
   go test -race ./...
   ```
6. **Commit** with a clear message describing the *why*
7. **Open a pull request** against `main`

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Keep functions short and focused
- Error messages include context: `fmt.Errorf("config: load %s: %w", path, err)`
- No global state — packages receive dependencies explicitly
- Table-driven tests with subtests

## Project Structure

```
cmd/mcpx/       entrypoint (thin — calls cli.Execute())
internal/
  config/       leaf package, zero internal deps
  secret/       leaf package, zero internal deps
  resolver/     depends on secret
  mcp/          zero internal deps (receives resolved strings)
  daemon/       depends on mcp, config
  cli/          orchestrator — wires everything together
```

**Rule:** `config` and `secret` are leaf packages. `cli` is the only place that imports multiple internal packages.

## What Makes a Good PR

- **One concern per PR** — a bug fix, a feature, or a refactor. Not all three.
- **Tests included** — if you add behavior, add a test. If you fix a bug, add a test that would have caught it.
- **No unrelated changes** — don't reformat files you didn't modify, don't add comments to code you didn't change.
- **Clear description** — explain what and why, not how (the code shows how).

## Reporting Bugs

Open an issue with:
- What you expected to happen
- What actually happened
- Steps to reproduce
- `mcpx version` output
- OS and Go version

## Feature Requests

Open an issue describing the use case. Focus on the problem you're solving, not the specific solution. We'll discuss the approach together.

## Security

If you find a security vulnerability, please report it privately. See [SECURITY.md](SECURITY.md) for details.

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
