# Security Policies

Policies are rules evaluated before every `tools/call` request. If a policy matches and the action is `deny`, the call is blocked before reaching the server.

## Policy Structure

```yaml
policies:
  - name: string           # unique name for error messages
    match:                  # what this policy matches
      tools: [string]       # tool name patterns (glob)
      args:                 # argument rules
        <arg-pattern>:      # arg name (supports glob like "*path*")
          deny_pattern: regex
          allow_prefix: [string]
          deny_prefix: [string]
      content:              # content inspection
        target: string      # e.g. "args.sql"
        deny_pattern: regex
        require_pattern: regex
        when: regex          # only apply when this matches
    action: string          # "allow", "deny", or "warn"
    message: string         # human-readable message on trigger
```

## Tool Name Matching

The `tools` field accepts glob patterns using `*` and `?`:

```yaml
# Match specific tools
tools: ["query", "execute"]

# Match patterns
tools: ["replace_*", "insert_*", "delete_*"]

# Match everything
tools: ["*"]
```

If `tools` is empty or omitted, the policy applies to all tools.

## Argument Rules

### `deny_pattern`

Block calls where an argument value matches a regex:

```yaml
args:
  relative_path:
    deny_pattern: "\\.\\.\\/|\\.\\.\\\\\\/"   # block path traversal
```

### `allow_prefix`

Require argument values to start with one of the listed prefixes:

```yaml
args:
  relative_path:
    allow_prefix: ["src/", "internal/", "cmd/", "./"]
```

If the value doesn't match any prefix, the policy triggers.

### `deny_prefix`

Block argument values that start with specific prefixes:

```yaml
args:
  relative_path:
    deny_prefix: ["../", "/etc/", "/root/"]
```

### Argument Name Globs

Argument names support glob patterns. This is useful for matching any path-like argument:

```yaml
args:
  "*path*":           # matches relative_path, file_path, etc.
    deny_pattern: "\\.\\.\\/|\\.\\.\\\\\\/"
```

## Content Match

For inspecting the body of arguments (e.g. SQL queries, code snippets):

```yaml
content:
  target: "args.sql"                              # which argument to inspect
  deny_pattern: "(?i)\\b(DROP|DELETE|TRUNCATE)\\b"  # block if matches
```

### `require_pattern` with `when`

Require a pattern to be present, but only when a condition matches:

```yaml
# Require LIMIT on SELECT queries
content:
  target: "args.sql"
  require_pattern: "(?i)\\bLIMIT\\b"
  when: "(?i)^\\s*SELECT"
```

This triggers the policy action when `when` matches but `require_pattern` doesn't.

## Global vs Server Policies

### Global policies

Apply to all servers. Defined under `security.global.policies`:

```yaml
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

### Server policies

Apply to a single server. Defined under `servers.<name>.security.policies`:

```yaml
servers:
  postgres:
    security:
      policies:
        - name: no-mutations
          match:
            tools: ["query"]
            content:
              target: args.sql
              deny_pattern: "(?i)\\b(INSERT|DELETE|DROP)\\b"
          action: deny
          message: "Mutation queries blocked"
```

### Evaluation Order

1. Global policies evaluated first (deny wins immediately)
2. Server `allowed_tools` / `blocked_tools` checked
3. Server policies evaluated
4. If nothing denied, the call proceeds

## Actions

| Action | Behavior |
|--------|----------|
| `deny` | Block the call. Return error to the agent. Log to audit. |
| `warn` | Print warning to stderr. Proceed with the call. Log to audit. |
| `allow` | Explicitly allow (useful in complex cascading setups). |

## Allow/Block Lists

For simple cases, use `allowed_tools` and `blocked_tools` instead of full policies:

```yaml
servers:
  jira:
    security:
      allowed_tools: [search_issues, get_issue, list_projects]
      blocked_tools: [delete_issue, close_issue]
```

- `allowed_tools`: Whitelist — only these tools can be called (glob patterns)
- `blocked_tools`: Blacklist — these tools are always denied (glob patterns)

If both are set, `allowed_tools` is checked first.
