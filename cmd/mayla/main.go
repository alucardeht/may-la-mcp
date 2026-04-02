package main

import (
	"log/slog"
	"os"

	"github.com/alucardeht/may-la-mcp/internal/mcp"
	"github.com/alucardeht/may-la-mcp/internal/tools"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})))

	registry := tools.NewRegistry()
	tools.RegisterAll(registry)

	server := mcp.NewServer(registry)
	if err := server.ProcessStream(os.Stdin, os.Stdout); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
