package daemon

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// IsRunning checks if a daemon is alive for the given server.
// It verifies both the PID file and socket connectivity.
func IsRunning(serverName string) bool {
	pidPath := PIDPath(serverName)
	socketPath := SocketPath(serverName)

	// Check PID file.
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return false
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false
	}

	// Check if process is alive.
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if !isProcessAlive(proc) {
		// Process is dead — clean up stale files.
		os.Remove(pidPath)
		os.Remove(socketPath)
		return false
	}

	// Verify socket is connectable.
	conn, err := net.DialTimeout("unix", socketPath, 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()

	return true
}

// EnsureRunning makes sure a daemon is running for the server.
// If not already running, it spawns one as a detached process.
// Returns the socket path to connect to.
func EnsureRunning(ctx context.Context, serverName string, command string, args []string, env []string, startupTimeout time.Duration) (string, error) {
	socketPath := SocketPath(serverName)

	if IsRunning(serverName) {
		return socketPath, nil
	}

	// Spawn a new daemon using ourselves with the hidden __daemon command.
	self, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("daemon: find executable: %w", err)
	}

	// Build daemon command args using base64-encoded JSON for safe transport.
	daemonArgs := []string{"__daemon", serverName, "--command", command}
	if len(args) > 0 {
		daemonArgs = append(daemonArgs, "--args", EncodeStringSlice(args))
	}
	if len(env) > 0 {
		daemonArgs = append(daemonArgs, "--env", EncodeStringSlice(env))
	}

	logPath := LogPath(serverName)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return "", fmt.Errorf("daemon: open log %s: %w", logPath, err)
	}

	// Use os.StartProcess for reliable detached process spawning on macOS.
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		logFile.Close()
		return "", fmt.Errorf("daemon: open devnull: %w", err)
	}

	fullArgs := append([]string{self}, daemonArgs...)
	attr := &os.ProcAttr{
		Dir:   "/",
		Files: []*os.File{devNull, logFile, logFile},
		Sys:   sysProcAttr(),
	}

	proc, err := os.StartProcess(self, fullArgs, attr)
	devNull.Close()
	logFile.Close()
	if err != nil {
		return "", fmt.Errorf("daemon: spawn: %w", err)
	}

	// Release the process so it outlives us.
	proc.Release()

	// Wait for the socket to become available.
	deadline := time.After(startupTimeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			return "", fmt.Errorf("daemon: %s failed to start within %s (check %s)", serverName, startupTimeout, logPath)
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			conn, err := net.DialTimeout("unix", socketPath, 500*time.Millisecond)
			if err == nil {
				conn.Close()
				return socketPath, nil
			}
		}
	}
}

// Stop sends SIGTERM to a running daemon.
func Stop(serverName string) error {
	pidPath := PIDPath(serverName)

	data, err := os.ReadFile(pidPath)
	if err != nil {
		return fmt.Errorf("daemon: %s not running (no pid file)", serverName)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		os.Remove(pidPath)
		return fmt.Errorf("daemon: corrupt pid file for %s", serverName)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(pidPath)
		return fmt.Errorf("daemon: process %d not found", pid)
	}

	if err := signalTerminate(proc); err != nil {
		// Process already dead — clean up.
		os.Remove(pidPath)
		os.Remove(SocketPath(serverName))
		return nil
	}

	// Wait briefly for cleanup.
	deadline := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			// Force kill.
			proc.Kill()
			os.Remove(pidPath)
			os.Remove(SocketPath(serverName))
			return nil
		case <-ticker.C:
			if !isProcessAlive(proc) {
				// Process exited.
				return nil
			}
		}
	}
}
