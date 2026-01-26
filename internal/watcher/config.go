package watcher

import "time"

type WatcherConfig struct {
	Enabled        bool          `json:"enabled"`
	DebounceWindow time.Duration `json:"debounce_window"`
	MaxBatchSize   int           `json:"max_batch_size"`
	IgnorePatterns []string      `json:"ignore_patterns"`
	WatchHidden    bool          `json:"watch_hidden"`
}

func DefaultWatcherConfig() WatcherConfig {
	return WatcherConfig{
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
	}
}
