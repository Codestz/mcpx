# Database Integrations

Secure configurations for database MCP servers.

## Postgres — Read-Only

Block all mutation queries while allowing reads:

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
          message: "Mutation queries blocked"

        - name: no-sensitive-tables
          match:
            tools: [query]
            content:
              target: args.sql
              deny_pattern: "(?i)\\b(credentials|api_keys|sessions|password_resets)\\b"
          action: deny
          message: "Access to sensitive tables is restricted"
```

### Usage

```bash
# Allowed
$ mcpx postgres query --sql "SELECT * FROM users LIMIT 10"

# Blocked
$ mcpx postgres query --sql "DROP TABLE users"
error: server "postgres": policy "no-mutations" denied tool "query"
  Reason: Mutation queries blocked
```

## MySQL

Same pattern works for MySQL MCP servers:

```yaml
servers:
  mysql:
    command: mysql-mcp
    env:
      MYSQL_URL: "$(secret.mysql_url)"
    security:
      policies:
        - name: no-mutations
          match:
            tools: [query]
            content:
              target: args.sql
              deny_pattern: "(?i)\\b(INSERT|UPDATE|DELETE|DROP|TRUNCATE|ALTER)\\b"
          action: deny
          message: "Mutation queries blocked"
```

## Require LIMIT on SELECT

Prevent full table scans with a warning policy:

```yaml
policies:
  - name: require-limit
    match:
      tools: [query]
      content:
        target: args.sql
        require_pattern: "(?i)\\bLIMIT\\b"
        when: "(?i)^\\s*SELECT"
    action: warn
    message: "SELECT without LIMIT — consider adding one"
```

This uses the `warn` action — the query still runs, but a warning is printed to stderr.

## Key Patterns

- Use `content.deny_pattern` for SQL keyword blocking
- Use `content.require_pattern` with `when` for conditional requirements
- Use `allowed_tools` to whitelist only query tools
- Use `blocked_tools` to deny raw execution tools
- Store connection strings in the OS keychain with `$(secret.*)`
