package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/alucardeht/may-la-mcp/internal/lsp"
	"github.com/alucardeht/may-la-mcp/internal/watcher"
)

type IndexConfig struct {
	Enabled         bool     `yaml:"enabled"`
	DBPath          string   `yaml:"db_path"`
	MaxFileSize     int64    `yaml:"max_file_size"`
	MaxQueueSize    int      `yaml:"max_queue_size"`
	WorkerCount     int      `yaml:"worker_count"`
	RateLimit       int      `yaml:"rate_limit"`
	ExcludePatterns []string `yaml:"exclude_patterns"`
}

type Config struct {
	DaemonAddr      string
	DaemonPort      int
	SocketPath      string
	DatabasePath    string
	LogLevel        string
	MaxConnections  int
	InstanceID      string
	InstanceDir     string
	Index           IndexConfig
	LSP             lsp.ManagerConfig `yaml:"lsp"`
	Watcher         watcher.WatcherConfig
}

func Load() *Config {
	homeDir, _ := os.UserHomeDir()
	maylaDir := filepath.Join(homeDir, ".mayla")
	socketPath := filepath.Join(maylaDir, "daemon.sock")
	dbPath := filepath.Join(maylaDir, "mayla.db")
	indexDBPath := filepath.Join(maylaDir, "index.db")

	return &Config{
		DaemonAddr:     "127.0.0.1",
		DaemonPort:     8765,
		SocketPath:     socketPath,
		DatabasePath:   dbPath,
		LogLevel:       "info",
		MaxConnections: 100,
		Index: IndexConfig{
			Enabled:      true,
			DBPath:       indexDBPath,
			MaxFileSize:  10 * 1024 * 1024,
			MaxQueueSize: 1000,
			WorkerCount:  2,
			RateLimit:    100,
			ExcludePatterns: []string{
				"**/node_modules/**",
				"**/.git/**",
				"**/vendor/**",
				"**/__pycache__/**",
				"**/target/**",
				"**/build/**",
				"**/dist/**",
			},
		},
		LSP: lsp.DefaultManagerConfig(),
		Watcher: watcher.WatcherConfig{
			Enabled:        true,
			DebounceWindow: 300 * time.Millisecond,
			MaxBatchSize:   100,
			IgnorePatterns: []string{
				"**/.git/**",
				"**/node_modules/**",
				"**/.idea/**",
				"**/*.log",
				"**/dist/**",
				"**/build/**",
				"**/__pycache__/**",
				"**/.venv/**",
				"**/vendor/**",
			},
			WatchHidden: false,
		},
	}
}

func (c *Config) EnsureDirectories() error {
	if c.InstanceDir != "" {
		return os.MkdirAll(c.InstanceDir, 0700)
	}
	homeDir, _ := os.UserHomeDir()
	maylaDir := filepath.Join(homeDir, ".mayla")
	return os.MkdirAll(maylaDir, 0700)
}

func LoadConfigWithInstance(instanceID string) (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	maylaDir := filepath.Join(homeDir, ".mayla")
	instanceDir := filepath.Join(maylaDir, "instances", instanceID)

	if err := os.MkdirAll(instanceDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create instance directory: %w", err)
	}

	return &Config{
		DaemonAddr:     "127.0.0.1",
		DaemonPort:     8765,
		SocketPath:     filepath.Join(instanceDir, "daemon.sock"),
		DatabasePath:   filepath.Join(instanceDir, "mayla.db"),
		LogLevel:       "info",
		MaxConnections: 100,
		InstanceID:     instanceID,
		InstanceDir:    instanceDir,
		Index: IndexConfig{
			Enabled:      true,
			DBPath:       filepath.Join(instanceDir, "index.db"),
			MaxFileSize:  10 * 1024 * 1024,
			MaxQueueSize: 1000,
			WorkerCount:  2,
			RateLimit:    100,
			ExcludePatterns: []string{
				"**/node_modules/**",
				"**/.git/**",
				"**/vendor/**",
				"**/__pycache__/**",
				"**/target/**",
				"**/build/**",
				"**/dist/**",
			},
		},
		LSP: lsp.DefaultManagerConfig(),
		Watcher: watcher.WatcherConfig{
			Enabled:        true,
			DebounceWindow: 300 * time.Millisecond,
			MaxBatchSize:   100,
			IgnorePatterns: []string{
				"**/.git/**",
				"**/node_modules/**",
				"**/.idea/**",
				"**/*.log",
				"**/dist/**",
				"**/build/**",
				"**/__pycache__/**",
				"**/.venv/**",
				"**/vendor/**",
			},
			WatchHidden: false,
		},
	}, nil
}
