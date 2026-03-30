# Security Overview

Every MCP tool call is an unrestricted operation. There's no authentication, no authorization, no audit trail between the AI agent and the server.

An agent connected to a Postgres MCP can `DROP TABLE` as easily as it can `SELECT`. A code search server can be pointed at `../../../etc/passwd`. A Slack MCP can post to `#general`.

**mcpx adds the missing security layer.**

## The Bridge

mcpx sits between every AI agent and every MCP server, evaluating security policies before any tool call reaches the server:

```
Agent (Claude, Cursor, Codex)
       │
       ▼
┌─────────────────────────┐
│  mcpx                   │
│  ├─ Security policies   │  ← decides WHAT you can do
│  ├─ Audit trail         │  ← records EVERYTHING
│  └─ Scoped daemons      │  ← isolates per project
└─────────────────────────┘
       │
       ▼
  MCP Servers (Serena, Postgres, Jira, Slack, etc.)
```

## Quick Example

Block path traversal on any MCP server:

```yaml
# .mcpx/config.yml
security:
  enabled: true
  global:
    policies:
      - name: no-path-traversal
        match:
          args:
            "*path*":
              deny_pattern: "\\.\\.\\/|\\.\\.\\\\\\/"
        action: deny
        message: "Path traversal blocked"
```

```bash
$ mcpx serena find_symbol --name "Config" --relative_path "../../../etc/passwd"

error: server "serena": policy "no-path-traversal" denied tool "find_symbol"
  Reason: Path traversal blocked
  relative_path = "../../../etc/passwd"
```

The call never reaches the server. The denial is logged to the audit trail.

## Three Layers

| Layer | What it does | Learn more |
|-------|-------------|------------|
| [Policies](/security/policies) | Tool allow/deny, argument inspection, content regex | Per-server and global rules |
| [Modes](/security/modes) | Quick presets: read-only, editing, custom | One line to restrict a server |
| [Audit Logging](/security/audit-logging) | JSONL log of every call with secret redaction | Compliance and debugging |

## Key Properties

- **Deny by default**: If a policy matches and the action is `deny`, the call is blocked before it reaches the server
- **Global + per-server**: Global policies apply everywhere, server policies add specificity
- **Zero overhead**: Policy evaluation is in-process, sub-millisecond
- **Protocol-level**: Works with any MCP server — security operates on tool names and arguments, not server internals
