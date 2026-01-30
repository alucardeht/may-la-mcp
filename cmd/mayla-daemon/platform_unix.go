//go:build unix

package main

import (
	"os"
	"os/signal"
	"syscall"
)

// processExists checks if a process with the given PID exists on Unix systems
// by sending signal 0 (which doesn't actually send a signal but checks if
// the process exists and we have permission to signal it)
func processExists(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// waitForShutdownSignal blocks until SIGINT or SIGTERM is received
func waitForShutdownSignal() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
}
