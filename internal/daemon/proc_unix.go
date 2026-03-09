//go:build !windows

package daemon

import (
	"os"
	"syscall"
)

func sysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setsid: true,
	}
}

// isProcessAlive checks if a process is running by sending signal 0.
func isProcessAlive(proc *os.Process) bool {
	return proc.Signal(syscall.Signal(0)) == nil
}

// signalTerminate sends SIGTERM to a process.
func signalTerminate(proc *os.Process) error {
	return proc.Signal(syscall.SIGTERM)
}
