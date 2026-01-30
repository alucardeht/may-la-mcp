package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/alucardeht/may-la-mcp/internal/config"
	"github.com/alucardeht/may-la-mcp/internal/daemon"
	"github.com/alucardeht/may-la-mcp/pkg/protocol"
)

const (
	readTimeout = 5 * time.Minute
)

var (
	instanceID  string
	daemonPID   int
	daemonCmd   *exec.Cmd
	instanceDir string
	cleanupOnce sync.Once
	daemonDone  chan struct{}
)

func main() {
	rand.Seed(time.Now().UnixNano())

	instanceID = generateInstanceID()
	daemonDone = make(chan struct{})

	cfg, err := config.LoadConfigWithInstance(instanceID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	instanceDir = cfg.InstanceDir

	setupCleanupHandlers()

	socketPath, existingHealthy := findExistingDaemon(cfg.SocketPath)
	if existingHealthy {
		log.Printf("Using existing daemon at %s\n", socketPath)
		daemonPID = -1
		daemonCmd = nil
	} else {
		pid, cmd, err := startDaemonForInstance(instanceID)
		daemonPID = pid
		daemonCmd = cmd
		if err != nil {
			cleanup()
			fmt.Fprintf(os.Stderr, "Failed to start daemon: %v\n", err)
			os.Exit(1)
		}
	}

	if err := waitForDaemonReady(cfg.SocketPath, 10*time.Second); err != nil {
		cleanup()
		fmt.Fprintf(os.Stderr, "Daemon failed to become ready: %v\n", err)
		os.Exit(1)
	}

	if daemonCmd != nil {
		go monitorDaemon(daemonCmd)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := connectWithRetry(ctx, cfg.SocketPath, 5)
	if err != nil {
		cleanup()
		fmt.Fprintf(os.Stderr, "Failed to connect to daemon: %v\n", err)
		os.Exit(1)
	}

	defer conn.Close()

	client := daemon.NewClient(conn)
	if err := handleStdio(ctx, client, cfg.SocketPath); err != nil {
		if ctx.Err() == nil {
			log.Printf("Error handling stdio: %v", err)
		}
	}

	cleanup()
}

func generateInstanceID() string {
	cwd, err := os.Getwd()
	if err == nil {
		hash := sha256.Sum256([]byte(cwd))
		hashHex := hex.EncodeToString(hash[:])
		return fmt.Sprintf("ws-%s", hashHex[:16])
	}

	return fmt.Sprintf("ws-%x", rand.Uint64())
}

func findExistingDaemon(socketPath string) (string, bool) {
	if _, err := os.Stat(socketPath); err != nil {
		return "", false
	}

	if isSocketHealthy(socketPath) {
		return socketPath, true
	}

	return "", false
}

func isSocketHealthy(socketPath string) bool {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return false
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(2 * time.Second))

	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params":  map[string]interface{}{},
	}

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(req); err != nil {
		return false
	}

	decoder := json.NewDecoder(conn)
	var resp map[string]interface{}
	if err := decoder.Decode(&resp); err != nil {
		return false
	}

	return true
}

func startDaemonForInstance(instanceID string) (int, *exec.Cmd, error) {
	execPath, err := os.Executable()
	if err != nil {
		return 0, nil, err
	}
	daemonPath := filepath.Join(filepath.Dir(execPath), "mayla-daemon")

	parentPID := os.Getpid()
	cmd := exec.Command(daemonPath, instanceID, fmt.Sprintf("%d", parentPID))
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return 0, nil, fmt.Errorf("failed to start daemon: %w", err)
	}

	return cmd.Process.Pid, cmd, nil
}

func waitForDaemonReady(socketPath string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if _, err := os.Stat(socketPath); err == nil {
			time.Sleep(100 * time.Millisecond)
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("daemon socket not ready after %v", timeout)
}

func monitorDaemon(cmd *exec.Cmd) {
	err := cmd.Wait()
	close(daemonDone)

	log.Printf("Daemon process exited: %v", err)
	cleanup()
	os.Exit(1)
}

func cleanup() {
	cleanupOnce.Do(func() {
		if daemonPID > 0 && daemonCmd != nil {
			killDaemon(daemonPID)
		}

		if instanceDir != "" && daemonPID > 0 {
			os.RemoveAll(instanceDir)
		}
	})
}

func connectWithRetry(ctx context.Context, socketPath string, maxRetries int) (net.Conn, error) {
	for i := 0; i < maxRetries; i++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		conn, err := connectToDaemon(socketPath)
		if err == nil {
			return conn, nil
		}

		if i < maxRetries-1 {
			backoffDuration := time.Duration(i+1) * 100 * time.Millisecond
			time.Sleep(backoffDuration)
		}
	}

	return nil, fmt.Errorf("failed to connect after %d retries", maxRetries)
}

func connectToDaemon(socketPath string) (net.Conn, error) {
	connector := daemon.NewSocketConnector(socketPath)
	return connector.Connect()
}

type stdinReader struct {
	decoder  *json.Decoder
	requests chan *protocol.JSONRPCRequest
	errors   chan error
	done     chan struct{}
}

func newStdinReader() *stdinReader {
	r := &stdinReader{
		decoder:  json.NewDecoder(os.Stdin),
		requests: make(chan *protocol.JSONRPCRequest, 10),
		errors:   make(chan error, 10),
		done:     make(chan struct{}),
	}

	go r.readLoop()

	return r
}

func (r *stdinReader) readLoop() {
	for {
		select {
		case <-r.done:
			return
		default:
		}

		var req protocol.JSONRPCRequest
		err := r.decoder.Decode(&req)

		if err != nil {
			select {
			case r.errors <- err:
			case <-r.done:
				return
			}
			continue
		}

		select {
		case r.requests <- &req:
		case <-r.done:
			return
		}
	}
}

func (r *stdinReader) readRequest(ctx context.Context) (*protocol.JSONRPCRequest, error) {
	select {
	case req := <-r.requests:
		return req, nil
	case err := <-r.errors:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (r *stdinReader) close() {
	close(r.done)
}

func handleStdio(ctx context.Context, client *daemon.Client, socketPath string) error {
	reader := newStdinReader()
	defer reader.close()

	writer := protocol.NewFlushWriter(os.Stdout)
	encoder := json.NewEncoder(writer)

	var writeMu sync.Mutex

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := reader.readRequest(ctx)
		if err != nil {
			if err == io.EOF || err == context.Canceled {
				return nil
			}
			return fmt.Errorf("failed to decode request: %w", err)
		}

		resp, err := client.SendRequest(req)
		if err != nil {
			if !client.IsHealthy() {
				log.Println("Connection unhealthy, attempting reconnect...")

				if err := client.Close(); err != nil {
					log.Printf("Error closing old connection: %v", err)
				}

				newConn, reconnErr := connectWithRetry(ctx, socketPath, 3)
				if reconnErr != nil {
					return fmt.Errorf("reconnection failed: %w", reconnErr)
				}

				client = daemon.NewClient(newConn)
				log.Println("Reconnected successfully")

				resp, err = client.SendRequest(req)
				if err != nil {
					return fmt.Errorf("request failed after reconnect: %w", err)
				}
			} else {
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
					if encodeErr == nil {
						writer.Flush()
					}
					writeMu.Unlock()
					if encodeErr != nil {
						return nil
					}
				}
				continue
			}
		}

		if req.ID != nil {
			writeMu.Lock()
			err := encoder.Encode(resp)
			if err == nil {
				writer.Flush()
			}
			writeMu.Unlock()
			if err != nil {
				return nil
			}
		}
	}
}
