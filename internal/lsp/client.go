package lsp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sourcegraph/jsonrpc2"
)

var (
	ErrNotInitialized = errors.New("lsp client not initialized")
	ErrAlreadyClosed  = errors.New("lsp client already closed")
	ErrTimeout        = errors.New("lsp request timeout")
)

type Client struct {
	conn         *jsonrpc2.Conn
	config       ClientConfig
	state        atomic.Value
	capabilities ServerCapabilities
	requestCount int64
	errorCount   int64
	lastRequest  time.Time
	mu           sync.RWMutex
	closedCh     chan struct{}
}

type ClientConfig struct {
	Language       Language
	InitTimeout    time.Duration
	RequestTimeout time.Duration
}

func DefaultClientConfig(lang Language) ClientConfig {
	return ClientConfig{
		Language:       lang,
		InitTimeout:    30 * time.Second,
		RequestTimeout: 10 * time.Second,
	}
}

type stdioReadWriteCloser struct {
	reader io.ReadCloser
	writer io.WriteCloser
}

func (s *stdioReadWriteCloser) Read(p []byte) (int, error) {
	return s.reader.Read(p)
}

func (s *stdioReadWriteCloser) Write(p []byte) (int, error) {
	return s.writer.Write(p)
}

func (s *stdioReadWriteCloser) Close() error {
	rerr := s.reader.Close()
	werr := s.writer.Close()
	if rerr != nil {
		return rerr
	}
	return werr
}

func NewClient(ctx context.Context, stdin io.WriteCloser, stdout io.ReadCloser, config ClientConfig) (*Client, error) {
	rwc := &stdioReadWriteCloser{
		reader: stdout,
		writer: stdin,
	}

	c := &Client{
		config:   config,
		closedCh: make(chan struct{}),
	}
	c.state.Store(StateStarting)

	stream := jsonrpc2.NewBufferedStream(rwc, jsonrpc2.VSCodeObjectCodec{})
	c.conn = jsonrpc2.NewConn(ctx, stream, &clientHandler{client: c})

	return c, nil
}

type clientHandler struct {
	client *Client
}

func (h *clientHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
}

func (c *Client) Initialize(ctx context.Context, rootURI string) error {
	c.mu.Lock()
	if c.getState() != StateStarting {
		c.mu.Unlock()
		return fmt.Errorf("cannot initialize: client in state %s", c.getState())
	}
	c.state.Store(StateInitializing)
	c.mu.Unlock()

	initCtx, cancel := context.WithTimeout(ctx, c.config.InitTimeout)
	defer cancel()

	params := InitializeParams{
		ProcessID: os.Getpid(),
		RootURI:   rootURI,
		Capabilities: map[string]interface{}{
			"textDocument": map[string]interface{}{
				"documentSymbol": map[string]interface{}{
					"hierarchicalDocumentSymbolSupport": true,
				},
			},
		},
	}

	var result InitializeResult
	if err := c.conn.Call(initCtx, "initialize", params, &result); err != nil {
		c.state.Store(StateError)
		return fmt.Errorf("initialize failed: %w", err)
	}

	c.mu.Lock()
	c.capabilities = result.Capabilities
	c.mu.Unlock()

	if err := c.conn.Notify(initCtx, "initialized", struct{}{}); err != nil {
		c.state.Store(StateError)
		return fmt.Errorf("initialized notification failed: %w", err)
	}

	c.state.Store(StateReady)
	return nil
}

func (c *Client) Shutdown(ctx context.Context) error {
	if !c.IsReady() {
		return ErrNotInitialized
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	var result interface{}
	if err := c.conn.Call(timeoutCtx, "shutdown", nil, &result); err != nil {
		return fmt.Errorf("shutdown failed: %w", err)
	}

	if err := c.conn.Notify(ctx, "exit", nil); err != nil {
		return fmt.Errorf("exit notification failed: %w", err)
	}

	return nil
}

func (c *Client) DocumentSymbols(ctx context.Context, uri string) ([]DocumentSymbol, error) {
	if !c.IsReady() {
		return nil, ErrNotInitialized
	}

	c.recordRequest()

	timeoutCtx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	params := DocumentSymbolParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	}

	var rawResult json.RawMessage
	if err := c.conn.Call(timeoutCtx, "textDocument/documentSymbol", params, &rawResult); err != nil {
		c.recordError()
		return nil, fmt.Errorf("documentSymbol request failed: %w", err)
	}

	var symbols []DocumentSymbol
	if err := json.Unmarshal(rawResult, &symbols); err == nil {
		return symbols, nil
	}

	var flatSymbols []SymbolInformation
	if err := json.Unmarshal(rawResult, &flatSymbols); err != nil {
		c.recordError()
		return nil, fmt.Errorf("failed to parse symbol response: %w", err)
	}

	return convertToDocumentSymbols(flatSymbols), nil
}

func convertToDocumentSymbols(flat []SymbolInformation) []DocumentSymbol {
	symbols := make([]DocumentSymbol, len(flat))
	for i, s := range flat {
		symbols[i] = DocumentSymbol{
			Name:           s.Name,
			Kind:           s.Kind,
			Range:          s.Location.Range,
			SelectionRange: s.Location.Range,
			Detail:         s.ContainerName,
		}
	}
	return symbols
}

func (c *Client) Close() error {
	select {
	case <-c.closedCh:
		return ErrAlreadyClosed
	default:
		close(c.closedCh)
	}

	c.state.Store(StateStopped)
	return c.conn.Close()
}

func (c *Client) IsReady() bool {
	return c.getState() == StateReady
}

func (c *Client) getState() LSPState {
	return c.state.Load().(LSPState)
}

func (c *Client) GetState() LSPState {
	return c.getState()
}

func (c *Client) Stats() ClientStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return ClientStats{
		Language:     c.config.Language,
		State:        c.getState(),
		RequestCount: atomic.LoadInt64(&c.requestCount),
		ErrorCount:   atomic.LoadInt64(&c.errorCount),
		LastRequest:  c.lastRequest,
	}
}

type ClientStats struct {
	Language     Language  `json:"language"`
	State        LSPState  `json:"state"`
	RequestCount int64     `json:"request_count"`
	ErrorCount   int64     `json:"error_count"`
	LastRequest  time.Time `json:"last_request,omitempty"`
}

func (c *Client) recordRequest() {
	atomic.AddInt64(&c.requestCount, 1)
	c.mu.Lock()
	c.lastRequest = time.Now()
	c.mu.Unlock()
}

func (c *Client) recordError() {
	atomic.AddInt64(&c.errorCount, 1)
}
