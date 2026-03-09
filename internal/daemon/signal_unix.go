//go:build !windows

package daemon

import (
	"os"
	"syscall"
)

// extraSignals returns platform-specific signals to listen for.
func extraSignals() []os.Signal {
	return []os.Signal{syscall.SIGTERM}
}
