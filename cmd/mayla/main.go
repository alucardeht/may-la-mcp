package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/alucardeht/may-la-mcp/internal/config"
	"github.com/alucardeht/may-la-mcp/internal/daemon"
	"github.com/alucardeht/may-la-mcp/pkg/protocol"
)

func main() {
	cfg := config.Load()

	conn, err := connectToDaemon(cfg.SocketPath)
	if err != nil {
		if err := startDaemon(cfg); err != nil {
			log.Fatalf("Failed to start daemon: %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		conn, err = connectToDaemon(cfg.SocketPath)
		if err != nil {
			log.Fatalf("Failed to connect to daemon: %v", err)
		}
	}

	defer conn.Close()

	client := daemon.NewClient(conn)
	if err := handleStdio(client); err != nil {
		log.Fatalf("Error handling stdio: %v", err)
	}
}

func connectToDaemon(socketPath string) (net.Conn, error) {
	connector := daemon.NewSocketConnector(socketPath)
	return connector.Connect()
}

func startDaemon(cfg *config.Config) error {
	if err := cfg.EnsureDirectories(); err != nil {
		return err
	}

	daemonPath := filepath.Join(os.Getenv("HOME"), ".mayla", "mayla-daemon")
	if _, err := os.Stat(daemonPath); err != nil {
		return fmt.Errorf("mayla-daemon not found at %s: %w", daemonPath, err)
	}

	cmd := exec.Command(daemonPath)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		waitTime := time.Duration(100*(i+1)) * time.Millisecond
		time.Sleep(waitTime)

		if _, err := os.Stat(cfg.SocketPath); err == nil {
			time.Sleep(100 * time.Millisecond)
			return nil
		}
	}

	return fmt.Errorf("daemon started but socket not created within %v seconds",
		time.Duration(100*(maxRetries*(maxRetries+1))/2)*time.Millisecond/time.Second)
}

func handleStdio(client *daemon.Client) error {
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		var req protocol.JSONRPCRequest
		if err := decoder.Decode(&req); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode request: %w", err)
		}

		resp, err := client.SendRequest(&req)
		if err != nil {
			errResp := &protocol.JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &protocol.JSONRPCError{
					Code:    -32603,
					Message: err.Error(),
				},
			}
			if err := encoder.Encode(errResp); err != nil {
				return err
			}
			continue
		}

		if err := encoder.Encode(resp); err != nil {
			return err
		}
	}

	return nil
}
