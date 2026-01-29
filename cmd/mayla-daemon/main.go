package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/alucardeht/may-la-mcp/internal/config"
	"github.com/alucardeht/may-la-mcp/internal/daemon"
	"github.com/alucardeht/may-la-mcp/internal/logger"
)

func init() {
	logCfg := logger.DefaultConfig()
	logCfg.Level = slog.LevelDebug
	logger.Init(logCfg)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Error: instance ID required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s <instance-id>\n", os.Args[0])
		os.Exit(1)
	}

	instanceID := os.Args[1]
	cfg, err := config.LoadConfigWithInstance(instanceID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.EnsureDirectories(); err != nil {
		log.Fatalf("Failed to ensure directories: %v", err)
	}

	d, err := daemon.NewDaemon(cfg)
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
