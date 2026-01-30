//go:build windows

package daemon

import (
	"syscall"
)

// processExists checks if a process with the given PID exists by attempting
// to open it with PROCESS_QUERY_LIMITED_INFORMATION access rights.
func processExists(pid int) bool {
	// PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	// This is the minimum access right needed to check if a process exists
	const processQueryLimitedInformation = 0x1000

	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return false
	}
	syscall.CloseHandle(h)
	return true
}
