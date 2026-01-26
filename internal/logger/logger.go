package logger

import (
	"io"
	"log/slog"
	"os"
)

type Config struct {
	Level     slog.Level
	Format    string
	Output    io.Writer
	AddSource bool
}

func DefaultConfig() Config {
	return Config{
		Level:     slog.LevelInfo,
		Format:    "text",
		Output:    os.Stderr,
		AddSource: false,
	}
}

func Init(cfg Config) {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level:     cfg.Level,
		AddSource: cfg.AddSource,
	}

	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(cfg.Output, opts)
	} else {
		handler = slog.NewTextHandler(cfg.Output, opts)
	}

	slog.SetDefault(slog.New(handler))
}

func Debug(msg string, args ...any) { slog.Debug(msg, args...) }
func Info(msg string, args ...any)  { slog.Info(msg, args...) }
func Warn(msg string, args ...any)  { slog.Warn(msg, args...) }
func Error(msg string, args ...any) { slog.Error(msg, args...) }

func ForComponent(component string) *slog.Logger {
	return slog.Default().With("component", component)
}

func With(args ...any) *slog.Logger {
	return slog.Default().With(args...)
}
