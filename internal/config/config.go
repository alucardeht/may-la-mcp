package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	DaemonAddr      string
	DaemonPort      int
	SocketPath      string
	DatabasePath    string
	LogLevel        string
	MaxConnections  int
}

func Load() *Config {
	homeDir, _ := os.UserHomeDir()
	socketPath := filepath.Join(homeDir, ".mayla", "daemon.sock")
	dbPath := filepath.Join(homeDir, ".mayla", "mayla.db")

	return &Config{
		DaemonAddr:     "127.0.0.1",
		DaemonPort:     8765,
		SocketPath:     socketPath,
		DatabasePath:   dbPath,
		LogLevel:       "info",
		MaxConnections: 100,
	}
}

func (c *Config) EnsureDirectories() error {
	homeDir, _ := os.UserHomeDir()
	maylaDir := filepath.Join(homeDir, ".mayla")
	return os.MkdirAll(maylaDir, 0700)
}
