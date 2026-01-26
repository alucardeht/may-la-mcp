package index

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alucardeht/may-la-mcp/internal/logger"
)

var log = logger.ForComponent("indexer")

type WorkerConfig struct {
	WorkerCount     int
	MaxQueueSize    int
	RateLimit       int
	MaxFileSize     int64
	ExcludePatterns []string
}

func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		WorkerCount:     2,
		MaxQueueSize:    1000,
		RateLimit:       100,
		MaxFileSize:     10 * 1024 * 1024,
		ExcludePatterns: []string{
			"**/node_modules/**",
			"**/.git/**",
			"**/vendor/**",
			"**/__pycache__/**",
			"**/target/**",
			"**/build/**",
			"**/dist/**",
		},
	}
}

type WorkerStats struct {
	Indexed     int64
	Failed      int64
	Skipped     int64
	InQueue     int64
	IsRunning   bool
	StartedAt   time.Time
	LastIndexed time.Time
}

type IndexWorker struct {
	store  *IndexStore
	config WorkerConfig

	highQueue   chan IndexJob
	normalQueue chan IndexJob
	lowQueue    chan IndexJob

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	rateLimiter *time.Ticker

	stats   WorkerStats
	statsMu sync.RWMutex
}

func NewIndexWorker(store *IndexStore, config WorkerConfig) *IndexWorker {
	ctx, cancel := context.WithCancel(context.Background())

	w := &IndexWorker{
		store:       store,
		config:      config,
		highQueue:   make(chan IndexJob, 100),
		normalQueue: make(chan IndexJob, config.MaxQueueSize),
		lowQueue:    make(chan IndexJob, config.MaxQueueSize*2),
		ctx:         ctx,
		cancel:      cancel,
	}

	if config.RateLimit > 0 {
		interval := time.Second / time.Duration(config.RateLimit)
		w.rateLimiter = time.NewTicker(interval)
	}

	return w
}

func (w *IndexWorker) Start() {
	w.statsMu.Lock()
	w.stats.IsRunning = true
	w.stats.StartedAt = time.Now()
	w.statsMu.Unlock()

	log.Info("index worker started", "workers", w.config.WorkerCount)

	for i := 0; i < w.config.WorkerCount; i++ {
		w.wg.Add(1)
		go w.worker(i)
	}
}

func (w *IndexWorker) Stop() {
	log.Info("index worker stopping")

	w.cancel()
	if w.rateLimiter != nil {
		w.rateLimiter.Stop()
	}
	w.wg.Wait()

	w.statsMu.Lock()
	w.stats.IsRunning = false
	w.statsMu.Unlock()

	log.Info("index worker stopped")
}

func (w *IndexWorker) Enqueue(job IndexJob) bool {
	var queue chan IndexJob
	switch job.Priority {
	case PriorityHigh:
		queue = w.highQueue
	case PriorityNormal:
		queue = w.normalQueue
	case PriorityLow:
		queue = w.lowQueue
	default:
		queue = w.normalQueue
	}

	if queue == nil {
		log.Error("CRITICAL: queue channel is nil!", "priority", job.Priority)
		return false
	}

	select {
	case queue <- job:
		atomic.AddInt64(&w.stats.InQueue, 1)
		return true
	default:
		log.Warn("job enqueue failed - queue full", "path", job.Path, "priority", job.Priority)
		return false
	}
}

func (w *IndexWorker) EnqueueBatch(paths []string, priority JobPriority) int {
	count := 0
	for _, path := range paths {
		if w.Enqueue(IndexJob{Path: path, Priority: priority}) {
			count++
		}
	}
	return count
}

func (w *IndexWorker) GetStats() WorkerStats {
	w.statsMu.RLock()
	defer w.statsMu.RUnlock()
	stats := w.stats
	stats.InQueue = atomic.LoadInt64(&w.stats.InQueue)
	return stats
}

func (w *IndexWorker) worker(id int) {
	defer w.wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			return
		default:
		}

		if w.rateLimiter != nil {
			select {
			case <-w.rateLimiter.C:
			case <-w.ctx.Done():
				return
			}
		}

		var job IndexJob
		var ok bool

		select {
		case job, ok = <-w.highQueue:
		default:
			select {
			case job, ok = <-w.normalQueue:
			default:
				select {
				case job, ok = <-w.lowQueue:
				default:
					time.Sleep(10 * time.Millisecond)
					continue
				}
			}
		}

		if !ok {
			continue
		}

		atomic.AddInt64(&w.stats.InQueue, -1)
		log.Debug("worker processing job", "worker_id", id, "path", job.Path)
		w.processJob(job)
	}
}

func (w *IndexWorker) processJob(job IndexJob) {
	path := job.Path
	log.Debug("processing file", "path", path)

	if w.shouldExclude(path) {
		w.recordSkipped()
		log.Debug("skipped file", "path", path, "reason", "excluded by pattern")
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		w.recordFailed(path, err.Error())
		log.Warn("failed to index", "path", path, "error", err)
		return
	}

	if info.IsDir() {
		return
	}

	if info.Size() > w.config.MaxFileSize {
		w.recordSkipped()
		w.store.UpdateFileStatus(path, StatusSkipped, "file too large")
		log.Debug("skipped file", "path", path, "reason", "file too large")
		return
	}

	existing, _ := w.store.GetFile(path)

	content, encoding, err := ReadFileAsUTF8(path)
	if err != nil {
		w.recordFailed(path, err.Error())
		log.Warn("failed to index", "path", path, "error", err)
		return
	}

	hash := sha256.Sum256([]byte(content))
	hashStr := hex.EncodeToString(hash[:])

	if existing != nil && existing.ContentHash == hashStr {
		log.Debug("skipped file", "path", path, "reason", "content unchanged")
		return
	}

	lang := detectLanguage(path)

	file := &IndexedFile{
		Path:        path,
		ContentHash: hashStr,
		Encoding:    encoding.Encoding,
		Language:    lang,
		Status:      StatusIndexed,
		IndexedAt:   time.Now(),
	}

	fileID, err := w.store.UpsertFile(file)
	if err != nil {
		w.recordFailed(path, err.Error())
		log.Warn("failed to index", "path", path, "error", err)
		return
	}

	symbols := extractSymbols(content, lang)
	if len(symbols) > 0 {
		if err := w.store.InsertSymbols(fileID, symbols); err != nil {
			w.recordFailed(path, err.Error())
			log.Warn("failed to index", "path", path, "error", err)
			return
		}
	}

	symbolCount := len(symbols)
	w.recordIndexed()
	log.Info("file indexed successfully", "path", path, "symbols", symbolCount)

	currentIndexed := atomic.LoadInt64(&w.stats.Indexed)
	if currentIndexed%100 == 0 {
		queueSize := atomic.LoadInt64(&w.stats.InQueue)
		log.Info("indexing progress", "indexed", currentIndexed, "pending", queueSize)
	}
}

func (w *IndexWorker) shouldExclude(path string) bool {
	for _, pattern := range w.config.ExcludePatterns {
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}

		if strings.Contains(pattern, "**") {
			parts := strings.Split(pattern, "**")
			for _, part := range parts {
				part = strings.Trim(part, "/")
				if part != "" && strings.Contains(path, "/"+part+"/") {
					return true
				}
			}
		}
	}
	return false
}

func (w *IndexWorker) recordIndexed() {
	atomic.AddInt64(&w.stats.Indexed, 1)
	w.statsMu.Lock()
	w.stats.LastIndexed = time.Now()
	w.statsMu.Unlock()
}

func (w *IndexWorker) recordFailed(path, errMsg string) {
	atomic.AddInt64(&w.stats.Failed, 1)
	w.store.UpdateFileStatus(path, StatusFailed, errMsg)
}

func (w *IndexWorker) recordSkipped() {
	atomic.AddInt64(&w.stats.Skipped, 1)
}

func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx", ".mjs":
		return "javascript"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".cxx", ".hpp":
		return "cpp"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".swift":
		return "swift"
	case ".kt", ".kts":
		return "kotlin"
	case ".scala":
		return "scala"
	case ".cs":
		return "csharp"
	default:
		return ""
	}
}

func extractSymbols(content, language string) []*IndexedSymbol {
	if language == "" {
		return nil
	}

	var patterns map[string]*regexp.Regexp
	switch language {
	case "go":
		patterns = goPatterns
	case "typescript", "javascript":
		patterns = tsPatterns
	case "python":
		patterns = pyPatterns
	case "java", "kotlin", "scala":
		patterns = javaPatterns
	case "rust":
		patterns = rustPatterns
	default:
		return nil
	}

	var symbols []*IndexedSymbol
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		for kind, re := range patterns {
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				name := matches[1]
				sym := &IndexedSymbol{
					Name:       name,
					Kind:       kind,
					LineStart:  lineNum + 1,
					LineEnd:    lineNum + 1,
					IsExported: isExported(name, language),
				}

				if len(matches) > 2 {
					sym.Signature = strings.TrimSpace(matches[0])
				}

				symbols = append(symbols, sym)
			}
		}
	}

	return symbols
}

func isExported(name, language string) bool {
	if name == "" {
		return false
	}
	switch language {
	case "go":
		return name[0] >= 'A' && name[0] <= 'Z'
	default:
		return !strings.HasPrefix(name, "_")
	}
}

var (
	goPatterns = map[string]*regexp.Regexp{
		"function":  regexp.MustCompile(`^\s*func\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`),
		"method":    regexp.MustCompile(`^\s*func\s+\([^)]+\)\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`),
		"type":      regexp.MustCompile(`^\s*type\s+([A-Za-z_][A-Za-z0-9_]*)\s+`),
		"interface": regexp.MustCompile(`^\s*type\s+([A-Za-z_][A-Za-z0-9_]*)\s+interface\s*\{`),
		"struct":    regexp.MustCompile(`^\s*type\s+([A-Za-z_][A-Za-z0-9_]*)\s+struct\s*\{`),
		"const":     regexp.MustCompile(`^\s*const\s+([A-Za-z_][A-Za-z0-9_]*)\s*`),
		"var":       regexp.MustCompile(`^\s*var\s+([A-Za-z_][A-Za-z0-9_]*)\s+`),
	}

	tsPatterns = map[string]*regexp.Regexp{
		"function":  regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?function\s+([A-Za-z_$][A-Za-z0-9_$]*)`),
		"class":     regexp.MustCompile(`^\s*(?:export\s+)?class\s+([A-Za-z_$][A-Za-z0-9_$]*)`),
		"interface": regexp.MustCompile(`^\s*(?:export\s+)?interface\s+([A-Za-z_$][A-Za-z0-9_$]*)`),
		"type":      regexp.MustCompile(`^\s*(?:export\s+)?type\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=`),
		"const":     regexp.MustCompile(`^\s*(?:export\s+)?const\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*[=:]`),
		"let":       regexp.MustCompile(`^\s*(?:export\s+)?let\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*[=:]`),
	}

	pyPatterns = map[string]*regexp.Regexp{
		"function": regexp.MustCompile(`^\s*def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`),
		"class":    regexp.MustCompile(`^\s*class\s+([A-Za-z_][A-Za-z0-9_]*)`),
		"method":   regexp.MustCompile(`^\s+def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`),
	}

	javaPatterns = map[string]*regexp.Regexp{
		"class":     regexp.MustCompile(`^\s*(?:public\s+)?(?:abstract\s+)?class\s+([A-Za-z_][A-Za-z0-9_]*)`),
		"interface": regexp.MustCompile(`^\s*(?:public\s+)?interface\s+([A-Za-z_][A-Za-z0-9_]*)`),
		"method":    regexp.MustCompile(`^\s*(?:public|private|protected)?\s*(?:static\s+)?[A-Za-z<>\[\]]+\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`),
	}

	rustPatterns = map[string]*regexp.Regexp{
		"function": regexp.MustCompile(`^\s*(?:pub\s+)?fn\s+([A-Za-z_][A-Za-z0-9_]*)`),
		"struct":   regexp.MustCompile(`^\s*(?:pub\s+)?struct\s+([A-Za-z_][A-Za-z0-9_]*)`),
		"enum":     regexp.MustCompile(`^\s*(?:pub\s+)?enum\s+([A-Za-z_][A-Za-z0-9_]*)`),
		"trait":    regexp.MustCompile(`^\s*(?:pub\s+)?trait\s+([A-Za-z_][A-Za-z0-9_]*)`),
		"impl":     regexp.MustCompile(`^\s*impl(?:<[^>]+>)?\s+([A-Za-z_][A-Za-z0-9_]*)`),
	}
)
