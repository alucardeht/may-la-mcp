package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/maylamcp/mayla/internal/config"
	"github.com/maylamcp/mayla/internal/daemon"
)

func main() {
	cfg := config.Load()
	if err := cfg.EnsureDirectories(); err != nil {
		log.Fatalf("Failed to ensure directories: %v", err)
	}

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
