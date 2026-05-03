package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// uiHandshake mirrors the file written by the UI daemon at ~/.mcpx/ui.json.
// Loaded by the gain command (and any other CLI surface that wants to surface
// the dashboard URL) without depending on the internal/ui package.
type uiHandshake struct {
	Port  int    `json:"port"`
	Token string `json:"token"`
	PID   int    `json:"pid"`
	Bind  string `json:"bind"`
}

func readUIHandshake(home string) (uiHandshake, error) {
	data, err := os.ReadFile(filepath.Join(home, ".mcpx", "ui.json"))
	if err != nil {
		return uiHandshake{}, err
	}
	var h uiHandshake
	if err := json.Unmarshal(data, &h); err != nil {
		return uiHandshake{}, err
	}
	if h.Bind == "" {
		h.Bind = "127.0.0.1"
	}
	return h, nil
}
