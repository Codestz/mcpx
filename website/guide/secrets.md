# Secrets

mcpx stores secrets in your OS keychain and injects them at runtime. Secrets never touch disk.

## Managing Secrets

### Store a secret

```bash
mcpx secret set github_token ghp_abc123def456
# ok  secret "github_token" stored. Use $(secret.github_token) in configs.
```

### List secrets

```bash
mcpx secret list
#   github_token  $(secret.github_token)
#   openai_key    $(secret.openai_key)
```

### Remove a secret

```bash
mcpx secret remove github_token
# Removed secret "github_token"
```

Aliases: `mcpx secret rm`, `mcpx secret delete`

## Using Secrets in Config

Reference secrets with `$(secret.<name>)` in your server config:

```yaml
servers:
  my-api:
    command: my-mcp-server
    env:
      API_KEY: "$(secret.api_key)"
      GITHUB_TOKEN: "$(secret.github_token)"
```

At runtime, mcpx resolves the variable from the keychain and injects it into the server process environment.

## Secret Name Rules

Names must:
- Contain only letters, digits, underscores, dashes, or dots
- Not start with a dot or dash

```bash
mcpx secret set my_api_key value       # valid
mcpx secret set github.token value     # valid
mcpx secret set my-key value           # valid
mcpx secret set .hidden value          # error
mcpx secret set "has space" value      # error
```

## JSON Output

```bash
mcpx secret list --json
# ["github_token", "openai_key"]

mcpx secret set mykey myval --json
# {"name": "mykey", "action": "set"}
```

## How It Works

mcpx uses the OS keychain via [go-keyring](https://github.com/zalando/go-keyring):

| OS | Backend |
|----|---------|
| macOS | Keychain |
| Linux | Secret Service (GNOME Keyring / KWallet) |
| Windows | Windows Credential Manager |

All secrets are stored under the service name `mcpx`. A metadata entry (`mcpx:keys`) tracks the list of stored secret names.

## Security Properties

- Secrets are resolved from the keychain at runtime
- They are injected into the server process environment, not passed as arguments
- They are never logged, cached, or written to any file
- They are never included in error messages
- `--dry-run` will show resolved values — be careful sharing output

## Troubleshooting

See [Secrets Errors](/troubleshooting/secrets-errors) for common issues with keychain access.
