//go:build unix

package daemon

import (
	"syscall"
)

// processExists checks if a process with the given PID exists by sending signal 0.
// On Unix systems, sending signal 0 to a process checks if it exists without
// actually sending a signal.
func processExists(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil
}
