# Communication Integrations

Secure configurations for Slack, email, and messaging MCP servers.

## Slack — Channel Restrictions

Allow the AI to read messages and search, but restrict where it can send:

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

### Usage

```bash
# Allowed
$ mcpx slack search_messages --query "deployment failed"
$ mcpx slack list_channels

# Blocked
$ mcpx slack send_message --channel "#general" --text "Hello"
error: server "slack": policy "no-public-channels" denied tool "send_message"
  Reason: Cannot post to public broadcast channels
```

## Email — Send Restrictions

```yaml
servers:
  email:
    command: email-mcp
    env:
      SMTP_URL: "$(secret.smtp_url)"
    security:
      mode: custom
      allowed_tools: [search_inbox, read_email, list_folders]
      blocked_tools: [delete_email, empty_trash]
      policies:
        - name: no-external
          match:
            tools: [send_email]
            args:
              to:
                deny_pattern: "^(?!.*@acme\\.com).*$"
          action: deny
          message: "Can only send to @acme.com addresses"
```

## Key Patterns

- Use `blocked_tools` to deny destructive operations (delete, edit)
- Use `deny_pattern` on channel/recipient arguments to restrict scope
- Use `warn` action for operations that should proceed but need visibility
- Separate read tools (`search_*`, `list_*`, `get_*`) from write tools (`send_*`, `post_*`)
- Store credentials in the OS keychain with `$(secret.*)`
