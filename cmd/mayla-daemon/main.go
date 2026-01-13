package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/alucardeht/may-la-mcp/internal/config"
	"github.com/alucardeht/may-la-mcp/internal/daemon"
)

func isDaemonAlreadyRunning(socketPath string) bool {
	conn, err := net.DialTimeout("unix", socketPath, 500*time.Millisecond)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

func cleanupStaleDaemonFiles(socketPath string) {
	if _, err := os.Stat(socketPath); err == nil {
		if !isDaemonAlreadyRunning(socketPath) {
			os.Remove(socketPath)
		}
	}
}

func main() {
	cfg := config.Load()
	if err := cfg.EnsureDirectories(); err != nil {
		log.Fatalf("Failed to ensure directories: %v", err)
	}

	if isDaemonAlreadyRunning(cfg.SocketPath) {
		fmt.Println("Daemon already running")
		os.Exit(0)
	}

	cleanupStaleDaemonFiles(cfg.SocketPath)

	d, err := daemon.NewDaemon(cfg.SocketPath)
	if err != nil {
		log.Fatalf("Failed to create daemon: %v", err)
	}

	if err := d.Start(); err != nil {
		log.Fatalf("Failed to start daemon: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("Shutting down daemon...")

	d.Shutdown()
}
