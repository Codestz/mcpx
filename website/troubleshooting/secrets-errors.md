# Secrets Errors

Problems with secret storage and resolution.

## Secret Not Found

```
error: server "serena": resolve env: secret get "github_token": secret not found
```

**Cause:** The config references `$(secret.github_token)` but no secret with that name is stored.

**Fix:**
```bash
mcpx secret set github_token ghp_your_actual_token
```

## Keychain Access Denied

```
error: secret set "mykey": The user name or passphrase you entered is not correct.
```

**Cause:** The OS denied access to the keychain.

**macOS:** You may need to unlock your keychain or grant access to the `mcpx` binary. See [Platform-Specific](/troubleshooting/platform-specific#macos-keychain-prompts).

**Linux:** You need a running Secret Service (GNOME Keyring or KWallet). See [Platform-Specific](/troubleshooting/platform-specific#linux-secret-service).

## Invalid Secret Name

```
error: secret name ".hidden" cannot start with "."
```

**Cause:** Secret names must follow naming rules.

**Rules:**
- Letters, digits, underscores, dashes, dots only
- Cannot start with `.` or `-`

```bash
# Valid
mcpx secret set api_key value
mcpx secret set github.token value
mcpx secret set my-key value

# Invalid
mcpx secret set .hidden value
mcpx secret set -flag value
mcpx secret set "has space" value
```

## Secret Resolves to Empty

If a secret exists but the server doesn't seem to receive it:

1. Verify the secret is stored:
   ```bash
   mcpx secret list
   ```

2. Use `--dry-run` to check resolution:
   ```bash
   mcpx serena search_symbol --name test --dry-run
   ```
   The dry-run output shows resolved environment variables.

3. Make sure the config uses the correct name:
   ```yaml
   env:
     TOKEN: "$(secret.github_token)"    # must match the name in mcpx secret set
   ```

## Keychain Corruption

If `mcpx secret list` shows errors or unexpected results:

The metadata key (`mcpx:keys`) in the keychain may be corrupted. You can reset it by deleting all mcpx secrets from your OS keychain directly:

**macOS:**
```bash
security delete-generic-password -s mcpx -a "mcpx:keys"
```

**Linux:**
Use `seahorse` (GNOME Keyring GUI) or `secret-tool` to find and delete entries with service `mcpx`.

Then re-add your secrets:
```bash
mcpx secret set mykey myvalue
```
