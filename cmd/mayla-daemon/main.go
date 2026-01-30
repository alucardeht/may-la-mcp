package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/alucardeht/may-la-mcp/internal/config"
	"github.com/alucardeht/may-la-mcp/internal/daemon"
	"github.com/alucardeht/may-la-mcp/internal/logger"
)

func init() {
	logCfg := logger.DefaultConfig()
	logCfg.Level = slog.LevelDebug
	logger.Init(logCfg)
}

func monitorParentProcess(ppid int, shutdownFunc func()) {
	log.Printf("Started monitoring parent process (PID: %d)", ppid)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if !processExists(ppid) {
			log.Println("Parent process died, triggering graceful shutdown")
			shutdownFunc()
			return
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Error: instance ID required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s <instance-id> [ppid]\n", os.Args[0])
		os.Exit(1)
	}

	instanceID := os.Args[1]

	var parentPID int
	if len(os.Args) >= 3 {
		ppid, err := strconv.Atoi(os.Args[2])
		if err == nil {
			parentPID = ppid
		}
	}

	homeDir, _ := os.UserHomeDir()
	logsDir := filepath.Join(homeDir, ".mayla", "logs")
	os.MkdirAll(logsDir, 0700)

	logFile := filepath.Join(logsDir, fmt.Sprintf("daemon-%s.log", instanceID))
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		log.SetOutput(io.MultiWriter(os.Stderr, f))
		defer f.Close()
	}

	cfg, err := config.LoadConfigWithInstance(instanceID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.EnsureDirectories(); err != nil {
		log.Fatalf("Failed to ensure directories: %v", err)
	}

	log.Printf("Daemon started for instance %s with workspace %s", instanceID, cfg.InstanceDir)

	d, err := daemon.NewDaemon(cfg)
	if err != nil {
		log.Fatalf("Failed to create daemon: %v", err)
	}

	if err := d.Start(); err != nil {
		log.Fatalf("Failed to start daemon: %v", err)
	}

	if parentPID > 0 {
		go monitorParentProcess(parentPID, func() {
			d.Shutdown()
			os.Exit(0)
		})
	}

	// Wait for shutdown signal (platform-specific)
	waitForShutdownSignal()

	log.Println("Shutting down daemon...")
	d.Shutdown()
}
