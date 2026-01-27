package lsp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/alucardeht/may-la-mcp/internal/logger"
)

var (
	ErrLanguageNotSupported = errors.New("language not supported")
	ErrNoProjectRoot        = errors.New("could not detect project root")
	ErrManagerClosed        = errors.New("manager is closed")

	log = logger.ForComponent("lsp")
)

type Manager struct {
	config    ManagerConfig
	processes map[Language]*Process
	starting  map[Language]bool

	idleTimers map[Language]*time.Timer
	lastAccess map[Language]time.Time

	mu       sync.RWMutex
	startMu  sync.Mutex
	closed   bool
	closedCh chan struct{}
}

func NewManager(config ManagerConfig) *Manager {
	return &Manager{
		config:     config,
		processes:  make(map[Language]*Process),
		starting:   make(map[Language]bool),
		idleTimers: make(map[Language]*time.Timer),
		lastAccess: make(map[Language]time.Time),
		closedCh:   make(chan struct{}),
	}
}

func (m *Manager) GetSymbols(ctx context.Context, path string) ([]DocumentSymbol, error) {
	if m.isClosed() {
		return nil, ErrManagerClosed
	}

	lang := m.DetectLanguage(path)
	if lang == "" {
		return nil, ErrLanguageNotSupported
	}

	serverConfig, ok := m.config.Servers[lang]
	if !ok || !serverConfig.Enabled {
		return nil, fmt.Errorf("%w: %s", ErrLanguageNotSupported, lang)
	}

	rootPath, found := m.FindProjectRoot(path, lang)
	if !found {
		rootPath = filepath.Dir(path)
	}

	process, err := m.getOrStartProcess(ctx, lang, rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get lsp process: %w", err)
	}

	m.recordAccess(lang)

	client := process.Client()
	if client == nil || !client.IsReady() {
		return nil, fmt.Errorf("lsp client not ready for %s", lang)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}
	uri := "file://" + absPath

	log.Debug("querying LSP for symbols", "path", path)

	symbols, err := client.DocumentSymbols(ctx, uri)
	if err != nil {
		return nil, err
	}

	log.Debug("LSP returned symbols", "path", path, "count", len(symbols))

	return symbols, nil
}

func (m *Manager) getOrStartProcess(ctx context.Context, lang Language, rootPath string) (*Process, error) {
	m.mu.RLock()
	if proc, exists := m.processes[lang]; exists {
		if proc.State() == StateReady && proc.RootPath() == rootPath {
			m.mu.RUnlock()
			return proc, nil
		}
	}
	m.mu.RUnlock()

	m.startMu.Lock()
	defer m.startMu.Unlock()

	m.mu.Lock()
	if proc, exists := m.processes[lang]; exists {
		if proc.State() == StateReady && proc.RootPath() == rootPath {
			m.mu.Unlock()
			return proc, nil
		}
		if oldProc := m.processes[lang]; oldProc != nil {
			if oldProc.State() == StateReady {
				stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				oldProc.Stop(stopCtx)
				cancel()
			}
			delete(m.processes, lang)
		}
	}

	if m.starting[lang] {
		m.mu.Unlock()
		return nil, fmt.Errorf("LSP for %s is already starting", lang)
	}
	m.starting[lang] = true
	m.mu.Unlock()

	serverConfig, ok := m.config.Servers[lang]
	if !ok {
		m.mu.Lock()
		delete(m.starting, lang)
		m.mu.Unlock()
		return nil, fmt.Errorf("no server configured for language: %s", lang)
	}

	proc := NewProcess(serverConfig)
	log.Info("starting LSP", "language", lang, "root", rootPath)
	err := proc.Start(ctx, rootPath)

	m.mu.Lock()
	delete(m.starting, lang)
	if err != nil {
		m.mu.Unlock()
		return nil, fmt.Errorf("failed to start LSP: %w", err)
	}
	m.processes[lang] = proc
	m.setupIdleTimer(lang)
	m.mu.Unlock()

	return proc, nil
}


func (m *Manager) stopProcessLocked(ctx context.Context, lang Language) error {
	proc, exists := m.processes[lang]
	if !exists {
		return nil
	}

	log.Info("stopping LSP", "language", lang, "reason", "idle")

	if timer, exists := m.idleTimers[lang]; exists {
		timer.Stop()
		delete(m.idleTimers, lang)
	}

	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := proc.Stop(stopCtx); err != nil {
		proc.Kill()
	}

	delete(m.processes, lang)
	delete(m.lastAccess, lang)

	return nil
}

func (m *Manager) setupIdleTimer(lang Language) {
	if timer, exists := m.idleTimers[lang]; exists {
		timer.Stop()
	}

	log.Debug("LSP idle timer set", "language", lang, "timeout", m.config.IdleTimeout)

	m.idleTimers[lang] = time.AfterFunc(m.config.IdleTimeout, func() {
		m.mu.Lock()
		defer m.mu.Unlock()

		if lastAccess, exists := m.lastAccess[lang]; exists {
			if time.Since(lastAccess) >= m.config.IdleTimeout {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				m.stopProcessLocked(ctx, lang)
			}
		}
	})
}

func (m *Manager) recordAccess(lang Language) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.lastAccess[lang] = time.Now()
	m.setupIdleTimer(lang)
}

func (m *Manager) GetProcess(lang Language) *Process {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.processes[lang]
}

func (m *Manager) StartProcess(ctx context.Context, lang Language, rootPath string) error {
	if m.isClosed() {
		return ErrManagerClosed
	}

	_, err := m.getOrStartProcess(ctx, lang, rootPath)
	return err
}

func (m *Manager) StopProcess(ctx context.Context, lang Language) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopProcessLocked(ctx, lang)
}

func (m *Manager) StopAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Info("stopping all LSP processes")

	var lastErr error
	for lang := range m.processes {
		if err := m.stopProcessLocked(ctx, lang); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}
	m.closed = true
	close(m.closedCh)

	log.Info("stopping all LSP processes")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var lastErr error
	for lang := range m.processes {
		if err := m.stopProcessLocked(ctx, lang); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (m *Manager) isClosed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closed
}

func (m *Manager) Stats() map[Language]LSPStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[Language]LSPStats)
	for lang, proc := range m.processes {
		stats[lang] = proc.Stats()
	}
	return stats
}

func (m *Manager) DetectLanguage(path string) Language {
	ext := strings.ToLower(filepath.Ext(path))

	for lang, config := range m.config.Servers {
		if !config.Enabled {
			continue
		}
		for _, e := range config.Extensions {
			if e == ext {
				return lang
			}
		}
	}

	return ""
}

func (m *Manager) FindProjectRoot(path string, lang Language) (string, bool) {
	config, ok := m.config.Servers[lang]
	if !ok {
		return "", false
	}

	dir := filepath.Dir(path)
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", false
	}

	for {
		for _, pattern := range config.RootPatterns {
			checkPath := filepath.Join(absDir, pattern)
			if _, err := os.Stat(checkPath); err == nil {
				return absDir, true
			}
		}

		parent := filepath.Dir(absDir)
		if parent == absDir {
			break
		}
		absDir = parent
	}

	return "", false
}

func (m *Manager) IsLanguageSupported(lang Language) bool {
	config, ok := m.config.Servers[lang]
	return ok && config.Enabled
}

func (m *Manager) IsLanguageInstalled(lang Language) bool {
	config, ok := m.config.Servers[lang]
	if !ok {
		return false
	}
	proc := NewProcess(config)
	return proc.IsInstalled()
}

func (m *Manager) EnabledLanguages() []Language {
	return m.config.GetEnabledLanguages()
}

func (m *Manager) InstalledLanguages() []Language {
	var installed []Language
	for lang, config := range m.config.Servers {
		if config.Enabled {
			proc := NewProcess(config)
			if proc.IsInstalled() {
				installed = append(installed, lang)
			}
		}
	}
	return installed
}
