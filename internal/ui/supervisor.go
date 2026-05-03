package ui

import (
	"net"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"
)

var ensureOnce sync.Once

// EnsureRunningAsync spawns the dashboard daemon if it isn't already running.
// Non-blocking: returns immediately and lets the daemon come up in background.
//
// Honors `MCPX_UI=off` to opt out.
func EnsureRunningAsync(enabled bool, configuredPort int, bind string) {
	if !enabled {
		return
	}
	if v := os.Getenv("MCPX_UI"); v == "off" || v == "0" || v == "false" {
		return
	}
	ensureOnce.Do(func() {
		if isAlive() {
			return
		}
		go spawn(configuredPort, bind)
	})
}

// isAlive returns true if the recorded daemon is reachable.
func isAlive() bool {
	h, err := LoadHandshake()
	if err != nil {
		return false
	}
	if h.PID > 0 {
		if proc, err := os.FindProcess(h.PID); err == nil {
			if err := proc.Signal(syscall.Signal(0)); err != nil {
				_ = RemoveHandshake()
				return false
			}
		}
	}
	addr := h.Bind + ":" + strconv.Itoa(h.Port)
	conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
	if err != nil {
		_ = RemoveHandshake()
		return false
	}
	conn.Close()
	return true
}

// spawn launches the UI daemon as a detached child process.
func spawn(port int, bind string) {
	self, err := os.Executable()
	if err != nil {
		return
	}
	args := []string{"__ui-run"}
	if port > 0 {
		args = append(args, "--port", strconv.Itoa(port))
	}
	if bind != "" {
		args = append(args, "--bind", bind)
	}

	cmd := exec.Command(self, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	cmd.SysProcAttr = detachAttr()
	_ = cmd.Start()
	if cmd.Process != nil {
		_ = cmd.Process.Release()
	}
}
