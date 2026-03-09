# Suggested Commands

## Build & Run
- go build ./cmd/mcpx          # build binary
- go build -o mcpx ./cmd/mcpx  # build with explicit output name
- ./mcpx version                # verify build
- go install ./cmd/mcpx        # install to GOPATH/bin

## Testing
- go test ./...                 # run all tests
- go test -race ./...           # run with race detector (always use this)
- go test ./internal/config/    # test single package
- go test -v -run TestName ./internal/cli/  # run specific test

## Linting & Vetting
- go vet ./...                  # static analysis
- go mod tidy                   # clean up dependencies

## mcpx CLI Usage
- ./mcpx list                   # list configured servers
- ./mcpx list <server>          # list tools for a server
- ./mcpx <server> <tool> --flags  # call a tool
- ./mcpx <server> <tool> --json   # JSON output mode
- ./mcpx <server> <tool> --dry-run # show what would execute
- ./mcpx init                   # import .mcp.json to .mcpx/config.yml
- ./mcpx daemon status          # show running daemons
- ./mcpx daemon stop <server>   # stop a daemon
- ./mcpx daemon stop-all        # stop all daemons

## Makefile Targets
- make build      # build binary
- make test       # run tests
- make lint       # go vet
- make clean      # remove binary
- make build-all  # cross-compile for all platforms