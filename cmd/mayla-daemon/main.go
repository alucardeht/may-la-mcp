package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/alucardeht/may-la-mcp/internal/config"
	"github.com/alucardeht/may-la-mcp/internal/daemon"
)

func main() {
	cfg := config.Load()
	if err := cfg.EnsureDirectories(); err != nil {
		log.Fatalf("Failed to ensure directories: %v", err)
	}

	pidPath := filepath.Join(filepath.Dir(cfg.SocketPath), "daemon.pid")

	if isAlreadyRunning(pidPath) {
		fmt.Println("Daemon already running")
		os.Exit(0)
	}

	if err := writePIDFile(pidPath); err != nil {
		log.Fatalf("Failed to write PID file: %v", err)
	}
	defer os.Remove(pidPath)

	if _, err := os.Stat(cfg.SocketPath); err == nil {
		os.Remove(cfg.SocketPath)
	}

	d, err := daemon.NewDaemon(cfg.SocketPath)
	if err != nil {
		log.Fatalf("Failed to create daemon: %v", err)
	}

	if err := d.Start(); err != nil {
		log.Fatalf("Failed to start daemon: %v", err)
	}

	handleSignals(d)
}

func isAlreadyRunning(pidPath string) bool {
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return false
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		os.Remove(pidPath)
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(pidPath)
		return false
	}

	if err := process.Signal(syscall.Signal(0)); err != nil {
		os.Remove(pidPath)
		return false
	}

	return true
}

func writePIDFile(pidPath string) error {
	return os.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())), 0644)
}

func handleSignals(d *daemon.Daemon) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	d.Shutdown()
}
