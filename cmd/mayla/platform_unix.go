//go:build unix

package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"
)

// setupCleanupHandlers sets up signal handlers for graceful shutdown on Unix systems
func setupCleanupHandlers() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		<-sigChan
		cleanup()
		os.Exit(0)
	}()
}

// killDaemon terminates the daemon process using Unix signals
func killDaemon(pid int) {
	// First try SIGTERM for graceful shutdown
	syscall.Kill(pid, syscall.SIGTERM)

	// Wait for process to exit, checking every 100ms for up to 5 seconds
	for i := 0; i < 50; i++ {
		if err := syscall.Kill(pid, 0); err != nil {
			return // Process has exited
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Force kill if still running
	syscall.Kill(pid, syscall.SIGKILL)
}
