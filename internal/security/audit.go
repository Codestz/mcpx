package security

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// AuditEntry represents a single audit log record.
type AuditEntry struct {
	Timestamp  string         `json:"timestamp"`
	Server     string         `json:"server"`
	Tool       string         `json:"tool"`
	Args       map[string]any `json:"args,omitempty"`
	Action     string         `json:"action"` // "allowed", "denied", "warned"
	PolicyName string         `json:"policy_name,omitempty"`
	Message    string         `json:"message,omitempty"`
}

// AuditLogger writes JSONL audit logs.
type AuditLogger struct {
	path    string
	redact  []string
	mu      sync.Mutex
}

// NewAuditLogger creates an audit logger that writes to the given path.
func NewAuditLogger(path string, redact []string) *AuditLogger {
	return &AuditLogger{
		path:   path,
		redact: redact,
	}
}

// Log writes an audit entry to the log file.
func (a *AuditLogger) Log(entry AuditEntry) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	entry.Timestamp = time.Now().UTC().Format(time.RFC3339)

	// Redact sensitive values.
	if len(a.redact) > 0 && entry.Args != nil {
		entry.Args = a.redactArgs(entry.Args)
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("audit: marshal: %w", err)
	}

	f, err := os.OpenFile(a.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("audit: open %s: %w", a.path, err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("audit: write: %w", err)
	}

	return nil
}

// redactArgs replaces values matching redact patterns with "[REDACTED]".
func (a *AuditLogger) redactArgs(args map[string]any) map[string]any {
	redacted := make(map[string]any, len(args))
	for k, v := range args {
		if a.shouldRedact(k, v) {
			redacted[k] = "[REDACTED]"
		} else {
			redacted[k] = v
		}
	}
	return redacted
}

// shouldRedact checks if a key or value matches a redact pattern.
func (a *AuditLogger) shouldRedact(key string, val any) bool {
	strVal := fmt.Sprintf("%v", val)
	for _, pattern := range a.redact {
		// Pattern like "$(secret.*)" — redact values that look like resolved secrets.
		if strings.Contains(pattern, "secret") {
			// Redact any arg whose key contains "secret", "token", "password", "key".
			lower := strings.ToLower(key)
			if strings.Contains(lower, "secret") ||
				strings.Contains(lower, "token") ||
				strings.Contains(lower, "password") ||
				strings.Contains(lower, "api_key") {
				return true
			}
		}
		// Also redact if the value itself starts with known secret prefixes.
		if strings.HasPrefix(strVal, "sk-") || strings.HasPrefix(strVal, "ghp_") {
			return true
		}
	}
	return false
}

// ActionString converts an Action to its audit log string.
func ActionString(a Action) string {
	switch a {
	case ActionAllow:
		return "allowed"
	case ActionDeny:
		return "denied"
	case ActionWarn:
		return "warned"
	default:
		return "unknown"
	}
}
