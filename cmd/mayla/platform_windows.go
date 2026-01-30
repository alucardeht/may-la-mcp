//go:build windows

package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"
)

// setupCleanupHandlers sets up signal handlers for graceful shutdown on Windows
func setupCleanupHandlers() {
	sigChan := make(chan os.Signal, 1)
	// Windows only supports os.Interrupt (Ctrl+C) and os.Kill
	signal.Notify(sigChan, os.Interrupt)

	go func() {
		<-sigChan
		cleanup()
		os.Exit(0)
	}()
}

// killDaemon terminates the daemon process on Windows using TerminateProcess
func killDaemon(pid int) {
	// Open the process with terminate permission
	const processTerminate = 0x0001
	const processQueryInformation = 0x0400

	h, err := syscall.OpenProcess(processTerminate|processQueryInformation, false, uint32(pid))
	if err != nil {
		return // Process may have already exited
	}
	defer syscall.CloseHandle(h)

	// Terminate the process with exit code 0
	err = syscall.TerminateProcess(h, 0)
	if err != nil {
		return
	}

	// Wait for process to exit, checking every 100ms for up to 5 seconds
	for i := 0; i < 50; i++ {
		checkH, err := syscall.OpenProcess(processQueryInformation, false, uint32(pid))
		if err != nil {
			return // Process has exited
		}
		syscall.CloseHandle(checkH)
		time.Sleep(100 * time.Millisecond)
	}
}
