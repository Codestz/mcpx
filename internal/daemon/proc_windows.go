//go:build windows

package daemon

import (
	"os"
	"syscall"
)

func sysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}

// isProcessAlive checks if a process is running.
// On Windows, Signal(0) is not supported so we try FindProcess.
func isProcessAlive(proc *os.Process) bool {
	// On Windows, FindProcess always succeeds. We try to signal —
	// if the process is gone, it returns an error.
	err := proc.Signal(os.Signal(syscall.Signal(0)))
	return err == nil
}

// signalTerminate kills the process on Windows (no SIGTERM support).
func signalTerminate(proc *os.Process) error {
	return proc.Kill()
}
