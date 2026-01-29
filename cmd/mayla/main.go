package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
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

	pid, cmd, err := startDaemonForInstance(instanceID)
	daemonPID = pid
	daemonCmd = cmd
	if err != nil {
		cleanup()
		fmt.Fprintf(os.Stderr, "Failed to start daemon: %v\n", err)
		os.Exit(1)
	}

	if err := waitForDaemonReady(cfg.SocketPath, 10*time.Second); err != nil {
		cleanup()
		fmt.Fprintf(os.Stderr, "Daemon failed to become ready: %v\n", err)
		os.Exit(1)
	}

	go monitorDaemon(cmd)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := connectToDaemon(cfg.SocketPath)
	if err != nil {
		cleanup()
		fmt.Fprintf(os.Stderr, "Failed to connect to daemon: %v\n", err)
		os.Exit(1)
	}

	defer conn.Close()

	client := daemon.NewClient(conn)
	if err := handleStdio(ctx, client); err != nil {
		if ctx.Err() == nil {
			log.Printf("Error handling stdio: %v", err)
		}
	}

	cleanup()
}

func generateInstanceID() string {
	return fmt.Sprintf("%d-%d-%x", os.Getpid(), time.Now().UnixNano(), rand.Int63())
}

func startDaemonForInstance(instanceID string) (int, *exec.Cmd, error) {
	execPath, err := os.Executable()
	if err != nil {
		return 0, nil, err
	}
	daemonPath := filepath.Join(filepath.Dir(execPath), "mayla-daemon")

	cmd := exec.Command(daemonPath, instanceID)
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

func setupCleanupHandlers() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		<-sigChan
		cleanup()
		os.Exit(0)
	}()
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
		if daemonPID > 0 {
			killDaemon(daemonPID)
		}

		if instanceDir != "" {
			os.RemoveAll(instanceDir)
		}
	})
}

func killDaemon(pid int) {
	syscall.Kill(pid, syscall.SIGTERM)

	for i := 0; i < 50; i++ {
		if err := syscall.Kill(pid, 0); err != nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	syscall.Kill(pid, syscall.SIGKILL)
}

func connectToDaemon(socketPath string) (net.Conn, error) {
	connector := daemon.NewSocketConnector(socketPath)
	return connector.Connect()
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

		select {
		case resultChan <- result{&req, err}:
		default:
		}
	}()

	if deadline, ok := r.ctx.Deadline(); ok {
		timeout := time.Until(deadline)
		timer := time.NewTimer(timeout)
		defer timer.Stop()

		select {
		case res := <-resultChan:
			return res.req, res.err
		case <-timer.C:
			return nil, context.DeadlineExceeded
		case <-r.ctx.Done():
			return nil, r.ctx.Err()
		}
	}

	timeoutTimer := time.NewTimer(r.timeout)
	defer timeoutTimer.Stop()

	select {
	case res := <-resultChan:
		return res.req, res.err
	case <-timeoutTimer.C:
		return nil, context.DeadlineExceeded
	case <-r.ctx.Done():
		return nil, r.ctx.Err()
	}
}

func handleStdio(ctx context.Context, client *daemon.Client) error {
	decoder := json.NewDecoder(os.Stdin)
	writer := protocol.NewFlushWriter(os.Stdout)
	encoder := json.NewEncoder(writer)

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
