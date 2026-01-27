package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/alucardeht/may-la-mcp/internal/config"
	"github.com/alucardeht/may-la-mcp/internal/daemon"
	"github.com/alucardeht/may-la-mcp/pkg/protocol"
)

const (
	readTimeout        = 30 * time.Second
	stdinCheckInterval = 5 * time.Second
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGPIPE)
	go func() {
		sig := <-sigChan
		log.Printf("CLI received signal %v, shutting down", sig)
		cancel()
	}()

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
	if err := handleStdio(ctx, client); err != nil {
		if ctx.Err() == nil {
			log.Printf("Error handling stdio: %v", err)
		}
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

	return fmt.Errorf("daemon started but socket not created")
}

type stdinReader struct {
	ctx     context.Context
	timeout time.Duration
}

func newStdinReader(ctx context.Context, timeout time.Duration) *stdinReader {
	return &stdinReader{
		ctx:     ctx,
		timeout: timeout,
	}
}

func (r *stdinReader) readRequest(decoder *json.Decoder) (*protocol.JSONRPCRequest, error) {
	type result struct {
		req *protocol.JSONRPCRequest
		err error
	}

	resultChan := make(chan result, 1)

	go func() {
		var req protocol.JSONRPCRequest
		err := decoder.Decode(&req)
		resultChan <- result{&req, err}
	}()

	ticker := time.NewTicker(stdinCheckInterval)
	defer ticker.Stop()

	timeoutTimer := time.NewTimer(r.timeout)
	defer timeoutTimer.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return nil, r.ctx.Err()

		case res := <-resultChan:
			if res.err != nil {
				return nil, res.err
			}
			return res.req, nil

		case <-ticker.C:
			if !isStdinValid() {
				return nil, io.EOF
			}

		case <-timeoutTimer.C:
			if !isStdinValid() {
				return nil, io.EOF
			}
			timeoutTimer.Reset(r.timeout)
		}
	}
}

func isStdinValid() bool {
	_, err := os.Stdin.Stat()
	return err == nil
}

func handleStdio(ctx context.Context, client *daemon.Client) error {
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	reader := newStdinReader(ctx, readTimeout)

	var writeMu sync.Mutex

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := reader.readRequest(decoder)
		if err != nil {
			if err == io.EOF || err == context.Canceled {
				return nil
			}
			return fmt.Errorf("failed to decode request: %w", err)
		}

		resp, err := client.SendRequest(req)
		if err != nil {
			if req.ID != nil {
				errResp := &protocol.JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error: &protocol.JSONRPCError{
						Code:    -32603,
						Message: err.Error(),
					},
				}
				writeMu.Lock()
				encodeErr := encoder.Encode(errResp)
				writeMu.Unlock()
				if encodeErr != nil {
					return nil
				}
			}
			continue
		}

		if req.ID != nil {
			writeMu.Lock()
			err := encoder.Encode(resp)
			writeMu.Unlock()
			if err != nil {
				return nil
			}
		}
	}
}
