# Project Management Integrations

Secure configurations for Jira, Linear, and similar project management MCP servers.

## Jira — Read-Only

Allow the AI to search and read issues, but not create, modify, or close them:

```yaml
servers:
  jira:
    command: jira-mcp
    transport: http
    url: "$(secret.jira_url)"
    auth:
      type: bearer
      token: "$(secret.jira_token)"
    security:
      mode: read-only
      allowed_tools: [search_issues, get_issue, list_projects, get_comments]
      policies:
        - name: restrict-projects
          match:
            tools: ["*"]
            args:
              project:
                allow_prefix: ["ENG-", "PLATFORM-"]
          action: deny
          message: "Access restricted to engineering projects"
```

### Usage

```bash
# Allowed
$ mcpx jira search_issues --query "type = Bug AND status = Open"
$ mcpx jira get_issue --key "ENG-1234"

# Blocked (read-only mode)
$ mcpx jira create_issue --project "ENG" --title "New bug"
error: server "jira": read-only mode denied tool "create_issue"
```

## Linear — Controlled Access

```yaml
servers:
  linear:
    command: linear-mcp
    env:
      LINEAR_API_KEY: "$(secret.linear_key)"
    security:
      mode: custom
      allowed_tools: [search_issues, get_issue, list_projects, list_labels]
      blocked_tools: [delete_issue, archive_project]
      policies:
        - name: restrict-teams
          match:
            tools: ["*"]
            args:
              team:
                allow_prefix: ["engineering", "platform"]
          action: deny
          message: "Access restricted to engineering teams"
```

## Key Patterns

- Use `mode: read-only` to block all write operations by default
- Use `allowed_tools` to whitelist specific read operations
- Use `allow_prefix` on project/team arguments to restrict scope
- Store API tokens in the OS keychain with `$(secret.*)`
- Use `transport: http` with `auth` for remote API-based servers
