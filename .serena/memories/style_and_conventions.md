# Code Style & Conventions

## Go Conventions
- Standard Go formatting (gofmt)
- Package-level errors wrapped with context: fmt.Errorf("config: load %s: %w", path, err)
- Sentinel errors for known conditions: ErrToolNotFound, ErrInitFailed, ErrTransportClosed
- Typed errors where useful: ToolError{Name, Message, Code}
- Table-driven tests with subtests: t.Run(name, func(t *testing.T) {...})
- Interfaces defined where consumed, not where implemented
- Only cli package calls os.Exit() - all others return errors

## Naming
- Package names are short, lowercase, single word
- Exported types: Config, ServerConfig, Client, Transport
- Constructor functions: NewClient, NewStdioTransport, NewKeyringStore
- No stutter: mcp.Client not mcp.MCPClient

## Testing
- Always run with -race flag
- Use testdata/ for fixtures
- Mock interfaces for unit tests (mockTransport, MemoryStore)
- Integration tests behind build tags when needed

## Error Handling
- Every package returns error, only cli calls os.Exit()
- Exit codes: 0=ok, 1=tool error, 2=config error, 3=connection error
- Errors wrapped with package prefix for traceability

## Config
- YAML format, two-level merge (global + project)
- Variable syntax: $(namespace.key) with strict regex validation
- Unknown namespaces are errors, not passthrough

## Security
- No shell expansion: exec.Command(cmd, args...) not sh -c
- Secrets resolved from keychain at runtime, never on disk
- Daemon sockets: unix socket mode 0600