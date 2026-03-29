# Security Examples

Real-world security configurations for common MCP servers.

## Postgres — Read-Only with SQL Filtering

```yaml
servers:
  postgres:
    command: postgres-mcp
    env:
      DATABASE_URL: "$(secret.pg_url)"
    security:
      mode: custom
      allowed_tools: [query, list_tables, describe_table, list_schemas]
      blocked_tools: [execute, create_database, drop_database]
      policies:
        - name: no-mutations
          match:
            tools: [query]
            content:
              target: args.sql
              deny_pattern: "(?i)\\b(INSERT|UPDATE|DELETE|DROP|TRUNCATE|ALTER|CREATE|GRANT|REVOKE)\\b"
          action: deny
          message: "Mutation queries blocked — this connection is read-only"

        - name: no-sensitive-tables
          match:
            tools: [query]
            content:
              target: args.sql
              deny_pattern: "(?i)\\b(credentials|api_keys|sessions|password_resets)\\b"
          action: deny
          message: "Access to sensitive tables is restricted"

        - name: require-limit
          match:
            tools: [query]
            content:
              target: args.sql
              require_pattern: "(?i)\\bLIMIT\\b"
              when: "(?i)^\\s*SELECT"
          action: warn
          message: "SELECT without LIMIT — consider adding one to avoid full table scans"
```

## Slack — Channel Restrictions

```yaml
servers:
  slack:
    command: slack-mcp
    transport: http
    url: "http://localhost:3001"
    security:
      mode: custom
      allowed_tools: [search_messages, list_channels, get_thread, get_channel_info]
      blocked_tools: [delete_message, edit_message]
      policies:
        - name: no-public-channels
          match:
            tools: [send_message]
            args:
              channel:
                deny_pattern: "^#general$|^#announcements$|^#all-hands$"
          action: deny
          message: "Cannot post to public broadcast channels"

        - name: no-dm-spam
          match:
            tools: [send_message]
            args:
              channel:
                deny_pattern: "^@"
          action: warn
          message: "Sending DMs — verify this is intended"
```

## Jira — Read-Only with Project Restrictions

```yaml
servers:
  jira:
    command: jira-mcp
    transport: http
    url: "$(secret.jira_url)"
    headers:
      Authorization: "Bearer $(secret.jira_token)"
    security:
      mode: read-only
      allowed_tools: [search_issues, get_issue, list_projects, get_comments]
      policies:
        - name: restrict-projects
          match:
            tools: ["*"]
            args:
              project:
                allow_prefix: ["ENG-", "PLATFORM-", "INFRA-"]
          action: deny
          message: "Access restricted to engineering projects"
```

## GitHub — Controlled Access

```yaml
servers:
  github:
    command: github-mcp-server
    env:
      GITHUB_TOKEN: "$(secret.github_token)"
    security:
      mode: custom
      allowed_tools: [search_*, get_*, list_*]
      blocked_tools: [create_*, delete_*, merge_*, close_*]
      policies:
        - name: restrict-repos
          match:
            tools: ["*"]
            args:
              repo:
                allow_prefix: ["acme/", "acme-platform/"]
          action: deny
          message: "Access restricted to acme organization repos"
```

## Serena — Path-Restricted Editing

```yaml
servers:
  serena:
    command: serena
    args: [start-mcp-server, --context=claude-code]
    daemon: true
    security:
      mode: editing
      policies:
        - name: source-only
          match:
            tools: ["replace_*", "insert_*", "rename_*"]
            args:
              relative_path:
                allow_prefix: ["src/", "internal/", "cmd/", "pkg/", "./"]
          action: deny
          message: "Writes restricted to source directories"

        - name: no-generated
          match:
            tools: ["replace_*", "insert_*"]
            args:
              relative_path:
                deny_prefix: ["generated/", "dist/", "build/", ".git/"]
          action: deny
          message: "Cannot modify generated or build artifacts"
```

## Filesystem — Directory Sandbox

```yaml
servers:
  filesystem:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem", "$(mcpx.project_root)"]
    security:
      mode: custom
      blocked_tools: [delete_file, move_file]
      policies:
        - name: no-dotfiles
          match:
            tools: ["*"]
            args:
              path:
                deny_pattern: "/\\."
          action: deny
          message: "Cannot access dotfiles or hidden directories"

        - name: no-escape
          match:
            tools: ["*"]
            args:
              path:
                deny_pattern: "\\.\\.\\/|\\.\\.\\\\\\/"
          action: deny
          message: "Cannot escape project directory"
```

## Combined Global + Per-Server

A typical team setup with global protections and server-specific rules:

```yaml
security:
  enabled: true
  global:
    audit:
      enabled: true
      log: "$(mcpx.project_root)/.mcpx/audit.jsonl"
      redact: ["$(secret.*)"]
    policies:
      - name: no-path-traversal
        match:
          args:
            "*path*":
              deny_pattern: "\\.\\.\\/|\\.\\.\\\\\\/"
        action: deny
        message: "Path traversal blocked"

servers:
  serena:
    security:
      mode: editing
  postgres:
    security:
      mode: read-only
  jira:
    security:
      mode: read-only
  slack:
    security:
      blocked_tools: [send_message, delete_message]
```
