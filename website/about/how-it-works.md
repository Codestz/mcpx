# How It Works

## Architecture

```
cmd/mcpx/              entrypoint
internal/
  config/              YAML config loading, merging, validation
  resolver/            dynamic variable resolution
  mcp/                 MCP protocol client (JSON-RPC 2.0 over stdio)
  cli/                 Cobra CLI, dynamic command generation
  daemon/              daemon process management, unix socket transport
  secret/              OS keychain integration
```

## Data Flow

What happens when you run `mcpx serena search_symbol --name "Auth"`:

```
1. Cobra parses → server="serena", tool="search_symbol", flags={name: "Auth"}
2. config.Load() → merged Config (global + project)
3. resolver.New() → resolves $(vars) in server config
4. Is daemon running?
   YES → connect via unix socket
   NO  → spawn subprocess, MCP handshake (or start daemon if daemon:true)
5. client.ListTools() → find "search_symbol" schema
6. Map CLI flags → tool arguments: {"name": "Auth"}
7. client.CallTool("search_symbol", {"name": "Auth"})
8. Format response → stdout
9. Close connection (daemon stays alive)
```

## MCP Protocol

mcpx implements the [Model Context Protocol](https://modelcontextprotocol.io) client side:

- **Transport**: JSON-RPC 2.0 over stdin/stdout (line-delimited), HTTP, or SSE
- **Handshake**: `initialize` request/response on first connection
- **Tools**: `tools/list` + `tools/call` — discover and invoke tools
- **Prompts**: `prompts/list` + `prompts/get` — discover and get prompt templates
- **Resources**: `resources/list` + `resources/templates/list` + `resources/read` — discover and read resources

### JSON-RPC Communication

```json
→ {"jsonrpc":"2.0","id":1,"method":"initialize","params":{...}}
← {"jsonrpc":"2.0","id":1,"result":{"capabilities":{...}}}

→ {"jsonrpc":"2.0","id":2,"method":"tools/list"}
← {"jsonrpc":"2.0","id":2,"result":{"tools":[...]}}

→ {"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search_symbol","arguments":{"name":"Auth"}}}
← {"jsonrpc":"2.0","id":3,"result":{"content":[{"type":"text","text":"..."}]}}
```

## Concurrency Model

### StdioTransport

- One reader goroutine demuxes responses by JSON-RPC ID
- `map[int64]chan *Response` routes responses to waiting callers (mutex-guarded)
- Writes serialized via mutex
- Atomic int64 for ID generation

### Daemon

- Separate OS process (survives CLI exit)
- Holds single MCP server subprocess
- Accepts connections on unix socket
- Per-connection goroutine reads requests, forwards to server, routes responses back
- Shutdown: SIGTERM → context cancellation → graceful close with 5s timeout

### Death Detection

When the MCP server subprocess crashes:
1. `readLoop` exits (stdout pipe closes)
2. `dead` channel is closed
3. Daemon detects transport death, calls shutdown
4. All pending connections receive errors and close
5. Socket and PID files are cleaned up
6. Next mcpx call starts a fresh daemon

## CLI Generation

mcpx dynamically generates CLI commands from MCP tool schemas:

| JSON Schema Type | Cobra Flag |
|------------------|------------|
| `string` | `StringVar` |
| `integer` | `Int64Var` |
| `number` | `Float64Var` |
| `boolean` | `BoolVar` |
| `array` | `StringVar` (parsed as JSON) |
| `object` | `StringVar` (parsed as JSON) |

Schema `required` fields become `MarkFlagRequired` in Cobra.

## Security Model

- **No shell expansion**: `exec.Command(cmd, args...)` — never `sh -c`
- **Strict variable regex**: `\$\(([a-z]+)\.([a-zA-Z0-9_.]+)\)`
- **Secrets from keychain only**: never on disk, never logged
- **Socket permissions**: `0600` (owner-only)
- **Subprocess isolation**: server stderr captured, never mixed with stdout
