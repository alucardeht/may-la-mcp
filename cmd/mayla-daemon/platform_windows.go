//go:build windows

package main

import (
	"os"
	"os/signal"
	"syscall"
)

// processExists checks if a process with the given PID exists on Windows
// by attempting to open the process handle
func processExists(pid int) bool {
	// PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	const processQueryLimitedInformation = 0x1000

	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return false
	}
	syscall.CloseHandle(h)
	return true
}

// waitForShutdownSignal blocks until an interrupt signal is received
func waitForShutdownSignal() {
	sigChan := make(chan os.Signal, 1)
	// Windows only supports os.Interrupt (Ctrl+C)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan
}
