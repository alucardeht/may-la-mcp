package lsp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrLSPNotInstalled  = errors.New("lsp server not installed")
	ErrMaxRestarts      = errors.New("max restart attempts exceeded")
	ErrProcessNotRunning = errors.New("process not running")
)

type Process struct {
	config   ServerConfig
	circuit  *CircuitBreaker

	cmd      *exec.Cmd
	client   *Client
	rootPath string

	state        atomic.Value
	restartCount int
	startedAt    time.Time
	lastError    error

	mu       sync.RWMutex
	stopOnce sync.Once
}

func NewProcess(config ServerConfig) *Process {
	p := &Process{
		config:  config,
		circuit: NewCircuitBreaker(DefaultCircuitConfig()),
	}
	p.state.Store(StateStopped)
	return p
}

func (p *Process) Start(ctx context.Context, rootPath string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	currentState := p.getState()
	if currentState == StateReady || currentState == StateStarting || currentState == StateInitializing {
		return nil
	}

	if p.restartCount >= p.config.MaxRestarts {
		return ErrMaxRestarts
	}

	if !p.circuit.Allow() {
		return fmt.Errorf("circuit breaker open for %s", p.config.Language)
	}

	path, err := exec.LookPath(p.config.Command)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrLSPNotInstalled, p.config.Command)
	}

	p.state.Store(StateStarting)
	p.rootPath = rootPath
	p.stopOnce = sync.Once{}

	p.cmd = exec.CommandContext(ctx, path, p.config.Args...)
	p.cmd.Dir = rootPath
	p.cmd.Env = append(os.Environ(),
		fmt.Sprintf("HOME=%s", os.Getenv("HOME")),
	)

	stdin, err := p.cmd.StdinPipe()
	if err != nil {
		p.state.Store(StateError)
		p.lastError = err
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdout, err := p.cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		p.state.Store(StateError)
		p.lastError = err
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := p.cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		p.state.Store(StateError)
		p.lastError = err
		p.circuit.RecordFailure()
		return fmt.Errorf("failed to start %s: %w", p.config.Command, err)
	}

	p.startedAt = time.Now()

	clientConfig := ClientConfig{
		Language:       p.config.Language,
		InitTimeout:    p.config.InitTimeout,
		RequestTimeout: p.config.RequestTimeout,
	}

	client, err := NewClient(ctx, stdin, stdout, clientConfig)
	if err != nil {
		p.killProcess()
		p.state.Store(StateError)
		p.lastError = err
		p.circuit.RecordFailure()
		return fmt.Errorf("failed to create lsp client: %w", err)
	}

	p.client = client

	rootURI := "file://" + rootPath
	if err := p.client.Initialize(ctx, rootURI); err != nil {
		p.killProcess()
		p.state.Store(StateError)
		p.lastError = err
		p.circuit.RecordFailure()
		p.restartCount++
		return fmt.Errorf("failed to initialize %s: %w", p.config.Language, err)
	}

	p.state.Store(StateReady)
	p.circuit.RecordSuccess()
	return nil
}

func (p *Process) Stop(ctx context.Context) error {
	var err error
	p.stopOnce.Do(func() {
		p.mu.Lock()
		defer p.mu.Unlock()

		if p.getState() == StateStopped {
			return
		}

		if p.client != nil && p.client.IsReady() {
			shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			if shutdownErr := p.client.Shutdown(shutdownCtx); shutdownErr != nil {
				err = shutdownErr
			}
			cancel()
			p.client.Close()
		}

		if p.cmd != nil && p.cmd.Process != nil {
			if sigErr := p.cmd.Process.Signal(os.Interrupt); sigErr != nil {
				err = sigErr
			}

			done := make(chan error, 1)
			go func() {
				done <- p.cmd.Wait()
			}()

			select {
			case <-done:
			case <-time.After(3 * time.Second):
				p.cmd.Process.Kill()
				<-done
			}
		}

		p.state.Store(StateStopped)
		p.client = nil
		p.cmd = nil
	})
	return err
}

func (p *Process) Kill() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd == nil || p.cmd.Process == nil {
		return nil
	}

	err := p.cmd.Process.Kill()
	p.state.Store(StateStopped)
	p.client = nil
	p.cmd = nil
	return err
}

func (p *Process) killProcess() {
	if p.cmd != nil && p.cmd.Process != nil {
		p.cmd.Process.Kill()
	}
	if p.client != nil {
		p.client.Close()
	}
	p.cmd = nil
	p.client = nil
}

func (p *Process) Client() *Client {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.client
}

func (p *Process) State() LSPState {
	return p.getState()
}

func (p *Process) getState() LSPState {
	return p.state.Load().(LSPState)
}

func (p *Process) Language() Language {
	return p.config.Language
}

func (p *Process) RootPath() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.rootPath
}

func (p *Process) Stats() LSPStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := LSPStats{
		Language: p.config.Language,
		State:    p.getState(),
	}

	if p.client != nil {
		clientStats := p.client.Stats()
		stats.RequestCount = clientStats.RequestCount
		stats.ErrorCount = clientStats.ErrorCount
		stats.LastRequest = clientStats.LastRequest
	}

	if !p.startedAt.IsZero() {
		stats.StartedAt = p.startedAt
		if p.getState() == StateReady {
			stats.Uptime = time.Since(p.startedAt)
		}
	}

	if p.lastError != nil {
		stats.LastErrorMsg = p.lastError.Error()
	}

	return stats
}

func (p *Process) CircuitState() CircuitState {
	return p.circuit.State()
}

func (p *Process) ResetCircuit() {
	p.circuit.Reset()
}

func (p *Process) ResetRestartCount() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.restartCount = 0
}

func (p *Process) IsInstalled() bool {
	_, err := exec.LookPath(p.config.Command)
	return err == nil
}

func (p *Process) Command() string {
	return p.config.Command
}
