//go:build windows

package main

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func setupCleanupHandlers() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	go func() {
		<-sigChan
		cleanup()
		os.Exit(0)
	}()
}

func killDaemon(pid int) {
	const processTerminate = 0x0001
	const processQueryInformation = 0x0400

	h, err := syscall.OpenProcess(processTerminate|processQueryInformation, false, uint32(pid))
	if err != nil {
		return
	}
	defer syscall.CloseHandle(h)

	err = syscall.TerminateProcess(h, 0)
	if err != nil {
		return
	}

	for i := 0; i < 50; i++ {
		checkH, err := syscall.OpenProcess(processQueryInformation, false, uint32(pid))
		if err != nil {
			break
		}
		syscall.CloseHandle(checkH)
		time.Sleep(100 * time.Millisecond)
	}

	cleanupStaleFiles()
}

func cleanupStaleFiles() {
	if instanceDir == "" {
		return
	}

	os.Remove(filepath.Join(instanceDir, "daemon.sock"))
	os.Remove(filepath.Join(instanceDir, "daemon.pid"))
	os.Remove(filepath.Join(instanceDir, "daemon.lock"))
}
