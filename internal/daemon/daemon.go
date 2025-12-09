package daemon

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/alucardeht/may-la-mcp/internal/mcp"
	"github.com/alucardeht/may-la-mcp/internal/tools"
	"github.com/alucardeht/may-la-mcp/internal/tools/files"
	"github.com/alucardeht/may-la-mcp/internal/tools/memory"
	"github.com/alucardeht/may-la-mcp/internal/tools/search"
	"github.com/alucardeht/may-la-mcp/internal/tools/spec"
)

type Daemon struct {
	socketPath    string
	listener      net.Listener
	registry      *tools.Registry
	server        *mcp.Server
	connections   map[net.Conn]bool
	connMu        sync.Mutex
	shutdown      chan struct{}
	shutdownOnce  sync.Once
	startTime     time.Time
}

func NewDaemon(socketPath string) (*Daemon, error) {
	d := &Daemon{
		socketPath:  socketPath,
		registry:    tools.NewRegistry(),
		connections: make(map[net.Conn]bool),
		shutdown:    make(chan struct{}),
		startTime:   time.Now(),
	}

	d.server = mcp.NewServer(d.registry)

	if err := d.registerAllTools(); err != nil {
		return nil, fmt.Errorf("failed to register tools: %w", err)
	}

	return d, nil
}

func (d *Daemon) registerAllTools() error {
	d.registry.Register(tools.NewHealthTool())

	for _, tool := range files.GetTools() {
		if err := d.registry.Register(tool); err != nil {
			return fmt.Errorf("files: %w", err)
		}
	}

	for _, tool := range search.GetTools() {
		if err := d.registry.Register(tool); err != nil {
			return fmt.Errorf("search: %w", err)
		}
	}

	for _, tool := range spec.GetTools() {
		if err := d.registry.Register(tool); err != nil {
			return fmt.Errorf("spec: %w", err)
		}
	}

	dataDir := filepath.Join(os.Getenv("HOME"), ".mayla", "data")
	os.MkdirAll(dataDir, 0755)
	dbPath := filepath.Join(dataDir, "memory.db")

	memTools, err := memory.GetTools(dbPath)
	if err != nil {
		return fmt.Errorf("memory: %w", err)
	}
	for _, tool := range memTools {
		if err := d.registry.Register(tool); err != nil {
			return fmt.Errorf("memory: %w", err)
		}
	}

	return nil
}

func (d *Daemon) Start() error {
	if err := os.RemoveAll(d.socketPath); err != nil {
		return fmt.Errorf("failed to remove socket: %w", err)
	}

	socketDir := filepath.Dir(d.socketPath)
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		return fmt.Errorf("failed to create socket dir: %w", err)
	}

	listener, err := net.Listen("unix", d.socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	d.listener = listener

	if err := os.Chmod(d.socketPath, 0700); err != nil {
		return fmt.Errorf("failed to chmod socket: %w", err)
	}

	go d.acceptConnections()
	d.handleSignals()

	return nil
}

func (d *Daemon) acceptConnections() {
	for {
		conn, err := d.listener.Accept()
		if err != nil {
			select {
			case <-d.shutdown:
				return
			default:
				continue
			}
		}

		d.connMu.Lock()
		d.connections[conn] = true
		d.connMu.Unlock()

		go d.handleConnection(conn)
	}
}

func (d *Daemon) handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		d.connMu.Lock()
		delete(d.connections, conn)
		d.connMu.Unlock()
	}()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		var req mcp.Request
		if err := decoder.Decode(&req); err != nil {
			return
		}

		resp := d.server.HandleRequest(&req)

		if err := encoder.Encode(resp); err != nil {
			return
		}
	}
}

func (d *Daemon) handleSignals() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	d.Shutdown()
}

func (d *Daemon) Shutdown() {
	d.shutdownOnce.Do(func() {
		close(d.shutdown)

		if d.listener != nil {
			d.listener.Close()
		}

		d.connMu.Lock()
		for conn := range d.connections {
			conn.Close()
		}
		d.connMu.Unlock()

		os.Remove(d.socketPath)
	})
}

func (d *Daemon) SocketPath() string {
	return d.socketPath
}

func (d *Daemon) Uptime() time.Duration {
	return time.Since(d.startTime)
}

func (d *Daemon) ToolCount() int {
	return len(d.registry.Names())
}
