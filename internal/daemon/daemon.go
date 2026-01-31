package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/alucardeht/may-la-mcp/internal/config"
	"github.com/alucardeht/may-la-mcp/internal/index"
	"github.com/alucardeht/may-la-mcp/internal/logger"
	"github.com/alucardeht/may-la-mcp/internal/lsp"
	"github.com/alucardeht/may-la-mcp/internal/mcp"
	"github.com/alucardeht/may-la-mcp/internal/router"
	"github.com/alucardeht/may-la-mcp/internal/tools"
	"github.com/alucardeht/may-la-mcp/internal/tools/docs"
	"github.com/alucardeht/may-la-mcp/internal/tools/files"
	"github.com/alucardeht/may-la-mcp/internal/tools/memory"
	"github.com/alucardeht/may-la-mcp/internal/tools/search"
	"github.com/alucardeht/may-la-mcp/internal/watcher"
	"github.com/alucardeht/may-la-mcp/pkg/protocol"
)

var log = logger.ForComponent("daemon")

type Daemon struct {
	socketPath     string
	listener       net.Listener
	registry       *tools.Registry
	server         *mcp.Server
	connections    map[net.Conn]bool
	connMu         sync.Mutex
	shutdown       chan struct{}
	shutdownOnce   sync.Once
	startTime      time.Time
	config         *config.Config
	indexStore     *index.IndexStore
	indexWorker    *index.IndexWorker
	lspManager     *lsp.Manager
	routerInstance *router.Router
	fileWatcher    *watcher.Watcher
	execSem        chan struct{}
	lifecycle      *LifecycleManager
	shuttingDown   atomic.Bool
	activeConns    sync.WaitGroup
	memoryStore    *memory.MemoryStore
}

func NewDaemon(cfg *config.Config) (*Daemon, error) {
	log.Info("initializing daemon", "socket", cfg.SocketPath)

	indexStore, err := index.NewIndexStore(cfg.Index.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create index store: %w", err)
	}
	log.Info("index store initialized", "path", cfg.Index.DBPath)

	indexWorkerConfig := index.WorkerConfig{
		WorkerCount:     cfg.Index.WorkerCount,
		MaxQueueSize:    cfg.Index.MaxQueueSize,
		RateLimit:       cfg.Index.RateLimit,
		MaxFileSize:     cfg.Index.MaxFileSize,
		ExcludePatterns: cfg.Index.ExcludePatterns,
	}
	indexWorker := index.NewIndexWorker(indexStore, indexWorkerConfig)
	log.Info("index worker initialized", "workers", cfg.Index.WorkerCount)

	lspManager := lsp.NewManager(cfg.LSP)
	log.Info("LSP manager initialized")

	routerInstance := router.NewRouter(indexStore, lspManager)
	log.Info("router initialized")

	watcherInstance, err := watcher.New(cfg.Watcher, indexWorker)
	if err != nil {
		indexStore.Close()
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}
	log.Info("watcher initialized")

	d := &Daemon{
		socketPath:     cfg.SocketPath,
		registry:       tools.NewRegistry(),
		connections:    make(map[net.Conn]bool),
		shutdown:       make(chan struct{}),
		startTime:      time.Now(),
		config:         cfg,
		indexStore:     indexStore,
		indexWorker:    indexWorker,
		lspManager:     lspManager,
		routerInstance: routerInstance,
		fileWatcher:    watcherInstance,
		execSem:        make(chan struct{}, 50),
		lifecycle:      NewLifecycleManager(filepath.Dir(cfg.SocketPath), cfg.SocketPath),
	}

	d.server = mcp.NewServer(d.registry)

	if err := d.registerAllTools(); err != nil {
		d.cleanupComponents()
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

	for _, tool := range docs.GetTools() {
		if err := d.registry.Register(tool); err != nil {
			return fmt.Errorf("docs: %w", err)
		}
	}

	for _, tool := range search.GetTools(d.routerInstance) {
		if err := d.registry.Register(tool); err != nil {
			return fmt.Errorf("search: %w", err)
		}
	}

	instanceDir := filepath.Dir(d.config.SocketPath)
	if err := os.MkdirAll(instanceDir, 0700); err != nil {
		return fmt.Errorf("failed to create instance directory: %w", err)
	}
	dbPath := filepath.Join(instanceDir, "memory.db")

	var err error
	d.memoryStore, err = memory.NewMemoryStore(dbPath)
	if err != nil {
		return fmt.Errorf("memory: %w", err)
	}

	memTools := memory.GetToolsFromStore(d.memoryStore)
	for _, tool := range memTools {
		if err := d.registry.Register(tool); err != nil {
			return fmt.Errorf("memory: %w", err)
		}
	}

	return nil
}

func (d *Daemon) Start() error {
	log.Info("daemon starting", "socket", d.socketPath)

	if err := d.lifecycle.AcquireInstanceLock(); err != nil {
		return fmt.Errorf("cannot start: %w", err)
	}

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

	if err := d.lifecycle.RegisterRunningDaemon(); err != nil {
		d.listener.Close()
		os.Remove(d.socketPath)
		return fmt.Errorf("failed to register daemon: %w", err)
	}

	if err := os.Chmod(d.socketPath, 0700); err != nil {
		d.lifecycle.Cleanup()
		d.listener.Close()
		os.Remove(d.socketPath)
		return fmt.Errorf("failed to chmod socket: %w", err)
	}

	log.Info("listening on socket", "path", d.socketPath)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-d.shutdown
		cancel()
	}()

	if d.config.Index.Enabled && d.indexWorker != nil {
		d.indexWorker.Start()
	}

	if d.config.Watcher.Enabled && d.fileWatcher != nil {
		if err := d.fileWatcher.Start(ctx); err != nil {
			log.Warn("failed to start watcher", "error", err)
		} else {
			cwd, err := os.Getwd()
			if err == nil {
				d.fileWatcher.AddRoot(cwd)
			}
		}
	}

	go d.acceptConnections()
	go d.handleSignals()

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

		d.activeConns.Add(1)
		log.Debug("accepted connection", "client", conn.RemoteAddr().String())
		go d.handleConnection(conn)
	}
}

func (d *Daemon) handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		d.connMu.Lock()
		delete(d.connections, conn)
		d.connMu.Unlock()
		d.activeConns.Done()
	}()

	writer := bufio.NewWriter(conn)
	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(writer)

	for {
		if err := conn.SetDeadline(time.Now().Add(5 * time.Minute)); err != nil {
			log.Error("failed to update connection deadline", "error", err)
			return
		}

		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return
		}

		if len(raw) == 0 {
			continue
		}

		if raw[0] == '[' {
			d.handleBatch(raw, encoder, writer)
		} else {
			d.handleSingleRequest(raw, encoder, writer)
		}
	}
}

func (d *Daemon) handleBatch(raw json.RawMessage, encoder *json.Encoder, writer *bufio.Writer) {
	var batch []mcp.Request
	if err := json.Unmarshal(raw, &batch); err != nil {
		errResp := &mcp.Response{
			JSONRPC: "2.0",
			ID:      nil,
			Error: &protocol.JSONRPCError{
				Code:    -32700,
				Message: "Parse error",
			},
		}
		if d.shuttingDown.Load() {
			return
		}
		if err := encoder.Encode(errResp); err != nil {
			log.Error("failed to encode parse error response", "error", err)
			return
		}
		if err := writer.Flush(); err != nil {
			log.Error("failed to flush parse error response", "error", err)
			return
		}
		return
	}

	select {
	case d.execSem <- struct{}{}:
		responses := d.server.HandleBatch(batch)
		<-d.execSem
		if d.shuttingDown.Load() {
			return
		}
		if err := encoder.Encode(responses); err != nil {
			log.Error("failed to encode batch responses", "error", err)
			return
		}
		if err := writer.Flush(); err != nil {
			log.Error("failed to flush batch responses", "error", err)
			return
		}
	case <-time.After(30 * time.Second):
		if d.shuttingDown.Load() {
			return
		}
		busyResps := make([]*mcp.Response, len(batch))
		for i, req := range batch {
			if req.ID != nil {
				busyResps[i] = &mcp.Response{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error: &protocol.JSONRPCError{
						Code:    -32603,
						Message: "server busy, try again later",
					},
				}
			}
		}
		if err := encoder.Encode(busyResps); err != nil {
			log.Error("failed to encode busy response", "error", err)
			return
		}
		if err := writer.Flush(); err != nil {
			log.Error("failed to flush busy response", "error", err)
			return
		}
	}
}

func (d *Daemon) handleSingleRequest(raw json.RawMessage, encoder *json.Encoder, writer *bufio.Writer) {
	var req mcp.Request
	if err := json.Unmarshal(raw, &req); err != nil {
		errResp := &mcp.Response{
			JSONRPC: "2.0",
			ID:      nil,
			Error: &protocol.JSONRPCError{
				Code:    -32700,
				Message: "Parse error",
			},
		}
		if d.shuttingDown.Load() {
			return
		}
		if err := encoder.Encode(errResp); err != nil {
			log.Error("failed to encode parse error response", "error", err)
			return
		}
		if err := writer.Flush(); err != nil {
			log.Error("failed to flush parse error response", "error", err)
			return
		}
		return
	}

	select {
	case d.execSem <- struct{}{}:
		resp := d.server.HandleRequest(&req)
		<-d.execSem
		if d.shuttingDown.Load() {
			return
		}
		if err := encoder.Encode(resp); err != nil {
			log.Error("failed to encode single response", "error", err)
			return
		}
		if err := writer.Flush(); err != nil {
			log.Error("failed to flush single response", "error", err)
			return
		}
	case <-time.After(30 * time.Second):
		if d.shuttingDown.Load() {
			return
		}
		busyResp := &mcp.Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &protocol.JSONRPCError{
				Code:    -32603,
				Message: "server busy, try again later",
			},
		}
		if err := encoder.Encode(busyResp); err != nil {
			log.Error("failed to encode busy response", "error", err)
			return
		}
		if err := writer.Flush(); err != nil {
			log.Error("failed to flush busy response", "error", err)
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
		log.Info("daemon shutting down")

		d.shuttingDown.Store(true)
		close(d.shutdown)

		done := make(chan struct{})
		go func() {
			d.activeConns.Wait()
			close(done)
		}()

		select {
		case <-done:
			log.Info("all connections drained gracefully")
		case <-time.After(30 * time.Second):
			log.Warn("shutdown timeout reached, forcing close")
		}

		d.cleanupComponents()

		if d.listener != nil {
			d.listener.Close()
		}

		d.connMu.Lock()
		for conn := range d.connections {
			conn.Close()
		}
		d.connMu.Unlock()

		os.Remove(d.socketPath)
		d.lifecycle.Cleanup()
		log.Info("daemon stopped")
	})
}

func (d *Daemon) cleanupComponents() {
	if d.fileWatcher != nil {
		d.fileWatcher.Stop()
	}

	if d.indexWorker != nil {
		d.indexWorker.Stop()
	}

	if d.lspManager != nil {
		d.lspManager.StopAll(context.Background())
	}

	if d.memoryStore != nil {
		if err := d.memoryStore.Close(); err != nil {
			log.Error("failed to close memory store", "error", err)
		}
	}

	if d.indexStore != nil {
		d.indexStore.Close()
	}
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
