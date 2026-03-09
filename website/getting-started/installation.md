# Installation

## Homebrew (macOS / Linux)

```bash
brew tap codestz/tap
brew install mcpx
```

## Go Install

Requires [Go 1.24+](https://go.dev/dl/):

```bash
go install github.com/codestz/mcpx/cmd/mcpx@latest
```

## Download Binary

Pre-built binaries for every release are available on the [GitHub Releases](https://github.com/codestz/mcpx/releases) page. Download the archive for your platform, extract, and move `mcpx` to your PATH.

## Verify

```bash
mcpx version
# mcpx 1.0.0
```

## Build from Source

```bash
git clone https://github.com/codestz/mcpx.git
cd mcpx
make build
```

The binary is built to `./mcpx`. Move it to your PATH:

```bash
sudo mv mcpx /usr/local/bin/
```

## Cross-Platform Builds

```bash
make build-all
```

Produces binaries in `dist/` for:
- `darwin-amd64` (macOS Intel)
- `darwin-arm64` (macOS Apple Silicon)
- `linux-amd64`
- `linux-arm64`
- `windows-amd64`

## Verify Installation

```bash
mcpx version          # print version
mcpx list             # should show "No servers configured" or your servers
```

## Next Steps

- [Quick Start](/getting-started/quick-start) — configure your first server
- [AI Agent Setup](/getting-started/ai-agent-setup) — wire mcpx into Claude Code or Cursor
