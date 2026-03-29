# Audit Logging

Every tool call that passes through mcpx can be recorded in a JSONL audit log — including allowed calls, denied calls, and warnings.

## Configuration

```yaml
security:
  enabled: true
  global:
    audit:
      enabled: true
      log: "$(mcpx.project_root)/.mcpx/audit.jsonl"
      redact: ["$(secret.*)"]
```

| Field | Description |
|-------|-------------|
| `enabled` | Enable or disable audit logging |
| `log` | Path to the JSONL log file. Supports [dynamic variables](/guide/variables). |
| `redact` | Patterns for values to redact. `$(secret.*)` redacts keys containing "secret", "token", "password", or "api_key", and values starting with `sk-` or `ghp_`. |

## Log Format

Each line is a JSON object:

```json
{"timestamp":"2026-03-28T18:19:50Z","server":"serena","tool":"find_symbol","args":{"name_path_pattern":"Config","relative_path":"internal/config/config.go"},"action":"allowed"}
```

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `timestamp` | string | ISO 8601 UTC timestamp |
| `server` | string | Server name from config |
| `tool` | string | Tool that was called |
| `args` | object | Arguments passed to the tool (redacted if configured) |
| `action` | string | `"allowed"`, `"denied"`, or `"warned"` |
| `policy_name` | string | Name of the policy that triggered (empty if allowed) |
| `message` | string | Policy message (empty if allowed) |

## Example Log

```jsonl
{"timestamp":"2026-03-28T18:19:50Z","server":"serena","tool":"find_symbol","args":{"name_path_pattern":"Config","relative_path":"internal/config/config.go"},"action":"allowed"}
{"timestamp":"2026-03-28T18:19:59Z","server":"serena","tool":"find_symbol","args":{"name_path_pattern":"Config","relative_path":"../../../etc/passwd"},"action":"denied","policy_name":"no-path-traversal","message":"Path traversal blocked"}
{"timestamp":"2026-03-28T18:20:10Z","server":"postgres","tool":"query","args":{"sql":"DROP TABLE users"},"action":"denied","policy_name":"no-mutations","message":"Mutation queries blocked"}
```

## Secret Redaction

When `redact: ["$(secret.*)"]` is configured, sensitive values are replaced with `[REDACTED]`:

```json
{"server":"postgres","tool":"connect","args":{"host":"localhost","password":"[REDACTED]","token":"[REDACTED]","name":"mydb"},"action":"allowed"}
```

Redaction targets:
- Argument keys containing `secret`, `token`, `password`, or `api_key`
- Argument values starting with `sk-` or `ghp_`

## Log Rotation

mcpx appends to the log file. For rotation, use standard tools:

```bash
# Rotate with logrotate, or simply:
mv .mcpx/audit.jsonl .mcpx/audit.$(date +%Y%m%d).jsonl
```

## Querying Logs

The JSONL format works with standard tools:

```bash
# Count denied calls
grep '"denied"' .mcpx/audit.jsonl | wc -l

# Find all calls by server
jq -r 'select(.server == "postgres")' .mcpx/audit.jsonl

# Find all denied calls with policy names
jq -r 'select(.action == "denied") | "\(.timestamp) \(.policy_name): \(.tool)"' .mcpx/audit.jsonl
```

## Gitignore

Add the audit log to `.gitignore`:

```txt
.mcpx/audit.jsonl
```

The `.mcpx/.gitignore` file is a good place for this if you want to keep it scoped:

```txt
# .mcpx/.gitignore
audit.jsonl
```
