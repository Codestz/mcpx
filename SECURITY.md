# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in mcpx, **please do not open a public issue.**

Instead, report it privately by emailing the maintainers or using [GitHub's private vulnerability reporting](https://github.com/codestz/mcpx/security/advisories/new).

Include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

We will acknowledge your report within 48 hours and work with you to understand and address the issue before any public disclosure.

## Security Model

mcpx is designed with the following security boundaries:

- **No shell expansion** — commands are executed via `exec.Command(cmd, args...)`, never through `sh -c`. There is no command injection surface.
- **Secrets never touch disk** — `$(secret.*)` variables are resolved from the OS keychain at runtime and injected into process environment. They are never logged, cached, or written to any file.
- **Strict variable parsing** — the variable pattern `$(namespace.key)` uses a strict regex. Unknown namespaces produce errors, not passthrough. No user-controlled string is ever interpreted as a shell command.
- **Daemon socket permissions** — unix sockets are created at `/tmp/mcpx-<server>-<uid>.sock` with mode `0600` (owner-only access).
- **Subprocess isolation** — MCP server stderr is captured and only displayed on error or `--dry-run`. It is never mixed with stdout.

## Supported Versions

| Version | Supported |
|---------|-----------|
| 1.x     | Yes       |
| < 1.0   | No        |
