package watcher

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/fsnotify/fsnotify"
	"github.com/alucardeht/may-la-mcp/internal/index"
	"github.com/alucardeht/may-la-mcp/internal/logger"
)

var log = logger.ForComponent("watcher")

type Watcher struct {
	config      WatcherConfig
	fsWatcher   *fsnotify.Watcher
	fsWatcherMu sync.Mutex
	debouncer   *Debouncer
	classifier  *EventClassifier
	indexer     *index.IndexWorker
	roots       []string
	mu          sync.RWMutex
	running     bool
	ctx         context.Context
	cancel      context.CancelFunc
}

func New(config WatcherConfig, indexer *index.IndexWorker) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		config:     config,
		fsWatcher:  fsWatcher,
		classifier: NewEventClassifier(),
		indexer:    indexer,
		roots:      make([]string, 0),
	}

	w.debouncer = NewDebouncer(config.DebounceWindow, config.MaxBatchSize, w.onFlush)

	return w, nil
}

func (w *Watcher) addToWatcher(path string) error {
	w.fsWatcherMu.Lock()
	defer w.fsWatcherMu.Unlock()
	return w.fsWatcher.Add(path)
}

func (w *Watcher) removeFromWatcher(path string) {
	w.fsWatcherMu.Lock()
	defer w.fsWatcherMu.Unlock()
	w.fsWatcher.Remove(path)
}

func (w *Watcher) AddRoot(path string) error {
	log.Info("adding root to watch", "path", path)

	if err := w.addToWatcher(path); err != nil {
		return err
	}

	w.mu.Lock()
	w.roots = append(w.roots, path)
	w.mu.Unlock()

	if err := w.walkAndAdd(path); err != nil {
		return err
	}

	log.Info("root added successfully", "path", path)
	return nil
}

func (w *Watcher) walkAndAdd(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		log.Debug("failed to read directory", "path", path, "error", err)
		return err
	}

	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())

		if w.shouldIgnore(fullPath) {
			continue
		}

		if entry.IsDir() {
			if err := w.addToWatcher(fullPath); err != nil {
				log.Debug("failed to watch directory", "path", fullPath, "error", err)
				continue
			}
			log.Debug("watching directory", "path", fullPath)
			w.walkAndAdd(fullPath)
		} else {
			if w.indexer == nil {
				log.Error("CRITICAL: indexer is nil!", "path", fullPath)
				continue
			}
			w.indexer.Enqueue(index.IndexJob{
				Path:     fullPath,
				Priority: index.PriorityLow,
			})
			log.Debug("enqueued file for indexing", "path", fullPath)
		}
	}

	return nil
}

func (w *Watcher) RemoveRoot(path string) error {
	w.removeFromWatcher(path)

	w.mu.Lock()
	defer w.mu.Unlock()

	for i, root := range w.roots {
		if root == path {
			w.roots = append(w.roots[:i], w.roots[i+1:]...)
			break
		}
	}

	return nil
}

func (w *Watcher) Start(ctx context.Context) error {
	log.Info("starting file watcher")

	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil
	}

	w.running = true
	w.ctx, w.cancel = context.WithCancel(ctx)
	w.mu.Unlock()

	go w.handleEvents()

	return nil
}

func (w *Watcher) handleEvents() {
	for {
		select {
		case <-w.ctx.Done():
			return

		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			log.Debug("file event", "path", event.Name, "op", event.Op.String())

			if event.Has(fsnotify.Create) {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					if !w.shouldIgnore(event.Name) {
						if err := w.addToWatcher(event.Name); err == nil {
							w.walkAndAdd(event.Name)
						}
					}
				}
			}

			fileEvent := w.convertEvent(event)
			if fileEvent != nil {
				w.debouncer.Add(*fileEvent)
			}

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}

			_ = err
		}
	}
}

func (w *Watcher) convertEvent(event fsnotify.Event) *FileEvent {
	if w.shouldIgnore(event.Name) {
		return nil
	}

	var eventType EventType

	switch {
	case event.Has(fsnotify.Create):
		eventType = EventCreate
	case event.Has(fsnotify.Write):
		eventType = EventModify
	case event.Has(fsnotify.Remove):
		eventType = EventDelete
	case event.Has(fsnotify.Rename):
		eventType = EventRename
	default:
		return nil
	}

	return &FileEvent{
		Path:      event.Name,
		Type:      eventType,
		Timestamp: time.Now(),
	}
}

func (w *Watcher) onFlush(events []FileEvent) {
	log.Info("flushing events", "count", len(events))

	if len(events) == 0 {
		return
	}

	if w.indexer == nil {
		log.Error("CRITICAL: indexer is nil in onFlush!")
		return
	}

	priority := w.classifier.ClassifyBatch(events)

	for _, event := range events {
		if event.Type == EventDelete {
			continue
		}

		job := index.IndexJob{
			Path:     event.Path,
			Priority: index.JobPriority(priority),
		}

		w.indexer.Enqueue(job)
	}
}

func (w *Watcher) shouldIgnore(path string) bool {
	basename := filepath.Base(path)

	if !w.config.WatchHidden && strings.HasPrefix(basename, ".") {
		return true
	}

	for _, pattern := range w.config.IgnorePatterns {
		if match, _ := doublestar.Match(pattern, path); match {
			return true
		}
	}

	return false
}

func (w *Watcher) Stop() error {
	log.Info("stopping file watcher")

	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return nil
	}

	w.running = false
	w.cancel()
	w.mu.Unlock()

	w.debouncer.Stop()

	w.fsWatcherMu.Lock()
	defer w.fsWatcherMu.Unlock()
	return w.fsWatcher.Close()
}

