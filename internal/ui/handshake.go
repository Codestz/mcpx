// Package ui hosts the always-on dashboard for mcpx.
//
// Lifecycle: any `mcpx <command>` invocation calls EnsureRunningAsync which
// lazily spawns a supervisor daemon. The daemon writes ~/.mcpx/ui.json with
// {port, token, pid, bind} so the CLI (and other tools) can discover the URL.
// Idle 1h with no HTTP traffic → daemon self-exits and removes the file.
package ui

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Handshake is what the UI daemon writes to ~/.mcpx/ui.json so other processes
// can discover the dashboard.
type Handshake struct {
	Port  int    `json:"port"`
	Token string `json:"token"`
	PID   int    `json:"pid"`
	Bind  string `json:"bind"`
}

// HandshakePath returns the on-disk handshake file path.
func HandshakePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".mcpx", "ui.json")
}

// LoadHandshake reads the handshake. Returns (zero, error) if missing.
func LoadHandshake() (Handshake, error) {
	path := HandshakePath()
	if path == "" {
		return Handshake{}, errors.New("ui: no home dir")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Handshake{}, err
	}
	var h Handshake
	if err := json.Unmarshal(data, &h); err != nil {
		return Handshake{}, err
	}
	return h, nil
}

// SaveHandshake writes the handshake atomically with mode 0600.
func SaveHandshake(h Handshake) error {
	path := HandshakePath()
	if path == "" {
		return errors.New("ui: no home dir")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// RemoveHandshake removes the handshake file. No-op if missing.
func RemoveHandshake() error {
	path := HandshakePath()
	if path == "" {
		return nil
	}
	err := os.Remove(path)
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}

// NewToken generates a 32-char hex token (128 bits).
func NewToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
