//go:build windows

package daemon

import "os"

// extraSignals returns platform-specific signals to listen for.
func extraSignals() []os.Signal {
	return nil
}
