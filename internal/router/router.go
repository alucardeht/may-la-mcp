package router

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/alucardeht/may-la-mcp/internal/index"
	"github.com/alucardeht/may-la-mcp/internal/logger"
	"github.com/alucardeht/may-la-mcp/internal/lsp"
)

var log = logger.ForComponent("router")

type Router struct {
	index      *index.IndexStore
	lspManager *lsp.Manager
	timeouts   TimeoutConfig
}

func NewRouter(indexStore *index.IndexStore, lspManager *lsp.Manager) *Router {
	return &Router{
		index:      indexStore,
		lspManager: lspManager,
		timeouts:   DefaultTimeoutConfig(),
	}
}

func NewRouterWithConfig(indexStore *index.IndexStore, lspManager *lsp.Manager, timeouts TimeoutConfig) *Router {
	return &Router{
		index:      indexStore,
		lspManager: lspManager,
		timeouts:   timeouts,
	}
}

func (r *Router) QuerySymbols(ctx context.Context, path string, query string, kinds []string, opts QueryOptions) (*QueryResult[Symbol], error) {
	start := time.Now()
	log.Debug("querying symbols", "path", path, "query", query)

	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	if !opts.SkipIndex && r.index != nil {
		log.Debug("trying index", "path", path)
		indexCtx, indexCancel := WithTimeout(ctx, r.timeouts.Index)
		result, err := r.queryIndexSymbols(indexCtx, path, query, kinds, opts)
		indexCancel()

		if err == nil && result != nil && len(result.Items) > 0 {
			fresh, err := IsFileFresh(r.index, path)
			if err != nil {
				fresh = false
			}
			if fresh {
				result.Latency = time.Since(start)
				result.Cached = true
				log.Debug("query completed", "source", result.Source, "count", result.Count, "latency_ms", result.Latency.Milliseconds())
				return result, nil
			}
		}
	}

	if !opts.SkipLSP && r.lspManager != nil {
		log.Debug("trying LSP", "path", path)
		lspCtx, lspCancel := WithTimeout(ctx, r.timeouts.LSP)
		result, err := r.queryLSPSymbols(lspCtx, path, query, kinds, opts)
		lspCancel()

		if err == nil && result != nil && len(result.Items) > 0 {
			result.Latency = time.Since(start)

			if opts.UpdateIndex && r.index != nil {
				r.updateIndexFromSymbols(path, result.Items)
			}

			log.Debug("query completed", "source", result.Source, "count", result.Count, "latency_ms", result.Latency.Milliseconds())
			return result, nil
		}
	}

	if opts.AllowFallback {
		log.Info("falling back to regex", "path", path, "reason", "index and LSP failed")
		regexCtx, regexCancel := WithTimeout(ctx, r.timeouts.Regex)
		result, err := r.queryRegexSymbols(regexCtx, path, query, kinds, opts)
		regexCancel()

		if err == nil {
			result.Latency = time.Since(start)
			result.Fallback = true
			log.Debug("query completed", "source", result.Source, "count", result.Count, "latency_ms", result.Latency.Milliseconds())
			return result, nil
		}
		return nil, err
	}

	return &QueryResult[Symbol]{
		Items:   []Symbol{},
		Count:   0,
		Source:  SourceIndex,
		Latency: time.Since(start),
	}, nil
}

func (r *Router) queryIndexSymbols(ctx context.Context, path string, query string, kinds []string, opts QueryOptions) (*QueryResult[Symbol], error) {
	file, err := r.index.GetFile(path)
	if err != nil || file == nil {
		return nil, err
	}

	indexed, err := r.index.GetSymbolsByFile(file.ID)
	if err != nil {
		return nil, err
	}

	var symbols []Symbol
	for _, s := range indexed {
		sym := FromIndexedSymbol(s)
		sym.File = path

		if query != "" && !strings.Contains(strings.ToLower(sym.Name), strings.ToLower(query)) {
			continue
		}

		if len(kinds) > 0 && !containsKind(kinds, sym.Kind) {
			continue
		}

		symbols = append(symbols, sym)

		if len(symbols) >= opts.MaxResults {
			break
		}
	}

	return &QueryResult[Symbol]{
		Items:  symbols,
		Count:  len(symbols),
		Source: SourceIndex,
	}, nil
}

func (r *Router) queryLSPSymbols(ctx context.Context, path string, query string, kinds []string, opts QueryOptions) (*QueryResult[Symbol], error) {
	lspSymbols, err := r.lspManager.GetSymbols(ctx, path)
	if err != nil {
		return nil, err
	}

	var symbols []Symbol
	flatSymbols := flattenLSPSymbols(lspSymbols, path)

	for _, sym := range flatSymbols {
		if query != "" && !strings.Contains(strings.ToLower(sym.Name), strings.ToLower(query)) {
			continue
		}

		if len(kinds) > 0 && !containsKind(kinds, sym.Kind) {
			continue
		}

		symbols = append(symbols, sym)

		if len(symbols) >= opts.MaxResults {
			break
		}
	}

	return &QueryResult[Symbol]{
		Items:  symbols,
		Count:  len(symbols),
		Source: SourceLSP,
	}, nil
}

func flattenLSPSymbols(symbols []lsp.DocumentSymbol, filePath string) []Symbol {
	var result []Symbol
	for _, s := range symbols {
		sym := Symbol{
			Name:      s.Name,
			Kind:      s.Kind.String(),
			File:      filePath,
			Line:      s.Range.Start.Line + 1,
			LineEnd:   s.Range.End.Line + 1,
			Column:    s.Range.Start.Character + 1,
			ColumnEnd: s.Range.End.Character + 1,
			Signature: s.Detail,
		}
		result = append(result, sym)

		if len(s.Children) > 0 {
			result = append(result, flattenLSPSymbols(s.Children, filePath)...)
		}
	}
	return result
}

func (r *Router) queryRegexSymbols(ctx context.Context, path string, query string, kinds []string, opts QueryOptions) (*QueryResult[Symbol], error) {
	content, _, err := index.ReadFileAsUTF8(path)
	if err != nil {
		return nil, err
	}

	lang := detectLanguage(path)
	if lang == "" {
		return &QueryResult[Symbol]{
			Items:  []Symbol{},
			Count:  0,
			Source: SourceRegex,
		}, nil
	}

	symbols := extractSymbolsRegex(content, path, lang, query, kinds, opts.MaxResults)

	return &QueryResult[Symbol]{
		Items:  symbols,
		Count:  len(symbols),
		Source: SourceRegex,
	}, nil
}

func (r *Router) updateIndexFromSymbols(path string, symbols []Symbol) {
	hasher := &FileHasher{}
	hash, err := hasher.ComputeHash(path)
	if err != nil {
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		return
	}

	file := &index.IndexedFile{
		Path:        path,
		ContentHash: hash,
		Language:    detectLanguage(path),
		Status:      index.StatusIndexed,
		IndexedAt:   time.Now(),
	}

	fileID, err := r.index.UpsertFile(file)
	if err != nil {
		return
	}

	var indexed []*index.IndexedSymbol
	for _, s := range symbols {
		indexed = append(indexed, &index.IndexedSymbol{
			Name:          s.Name,
			Kind:          s.Kind,
			Signature:     s.Signature,
			LineStart:     s.Line,
			LineEnd:       s.LineEnd,
			ColumnStart:   s.Column,
			ColumnEnd:     s.ColumnEnd,
			Documentation: s.Documentation,
			IsExported:    s.IsExported,
		})
	}

	r.index.InsertSymbols(fileID, indexed)
	_ = info
}

func (r *Router) QueryReferences(ctx context.Context, symbol string, path string, opts QueryOptions) (*QueryResult[Reference], error) {
	start := time.Now()
	log.Debug("querying references", "symbol", symbol, "path", path)

	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	if !opts.SkipIndex && r.index != nil {
		log.Debug("trying index", "path", path)
		indexCtx, indexCancel := WithTimeout(ctx, r.timeouts.Index)
		result, err := r.queryIndexReferences(indexCtx, symbol, path, opts)
		indexCancel()

		if err == nil && result != nil && len(result.Items) > 0 {
			result.Latency = time.Since(start)
			log.Debug("references found", "source", result.Source, "count", result.Count)
			return result, nil
		}
	}

	if opts.AllowFallback {
		log.Info("falling back to regex", "path", path, "reason", "index failed")
		regexCtx, regexCancel := WithTimeout(ctx, r.timeouts.Regex)
		result, err := r.queryRegexReferences(regexCtx, symbol, path, opts)
		regexCancel()

		if err == nil {
			result.Latency = time.Since(start)
			result.Fallback = true
			log.Debug("references found", "source", result.Source, "count", result.Count)
			return result, nil
		}
		return nil, err
	}

	return &QueryResult[Reference]{
		Items:   []Reference{},
		Count:   0,
		Source:  SourceIndex,
		Latency: time.Since(start),
	}, nil
}

func (r *Router) queryIndexReferences(ctx context.Context, symbol string, path string, opts QueryOptions) (*QueryResult[Reference], error) {
	indexed, err := r.index.SearchSymbols(symbol, opts.MaxResults)
	if err != nil || len(indexed) == 0 {
		return nil, err
	}

	refs, err := r.index.GetReferencesForSymbol(indexed[0].ID)
	if err != nil {
		return nil, err
	}

	var references []Reference
	for _, ref := range refs {
		file, _ := r.index.GetFileByID(ref.FileID)
		filePath := path
		if file != nil {
			filePath = file.Path
		}

		reference := FromIndexedReference(ref)
		reference.File = filePath
		references = append(references, reference)

		if len(references) >= opts.MaxResults {
			break
		}
	}

	return &QueryResult[Reference]{
		Items:  references,
		Count:  len(references),
		Source: SourceIndex,
	}, nil
}

func (r *Router) queryRegexReferences(ctx context.Context, symbol string, searchPath string, opts QueryOptions) (*QueryResult[Reference], error) {
	var references []Reference

	pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(symbol) + `\b`)

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		lang := detectLanguage(path)
		if lang == "" {
			return nil
		}

		content, _, err := index.ReadFileAsUTF8(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(content, "\n")
		for lineNum, line := range lines {
			if pattern.MatchString(line) {
				loc := pattern.FindStringIndex(line)
				col := 0
				if loc != nil {
					col = loc[0] + 1
				}

				references = append(references, Reference{
					File:    path,
					Line:    lineNum + 1,
					Column:  col,
					Context: strings.TrimSpace(line),
					Kind:    classifyReference(line, symbol),
				})

				if len(references) >= opts.MaxResults {
					return filepath.SkipAll
				}
			}
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return nil, err
	}

	return &QueryResult[Reference]{
		Items:  references,
		Count:  len(references),
		Source: SourceRegex,
	}, nil
}

func containsKind(kinds []string, kind string) bool {
	for _, k := range kinds {
		if strings.EqualFold(k, kind) {
			return true
		}
	}
	return false
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
	default:
		return ""
	}
}

func classifyReference(line, symbol string) string {
	lower := strings.ToLower(line)
	if strings.Contains(lower, "import") || strings.Contains(lower, "require") {
		return "import"
	}
	if strings.Contains(lower, "func ") || strings.Contains(lower, "function ") ||
		strings.Contains(lower, "def ") || strings.Contains(lower, "class ") {
		return "definition"
	}
	return "usage"
}

func extractSymbolsRegex(content, filePath, lang, query string, kinds []string, maxResults int) []Symbol {
	var symbols []Symbol
	lines := strings.Split(content, "\n")

	patterns := getLanguagePatterns(lang)
	if patterns == nil {
		return symbols
	}

	for lineNum, line := range lines {
		for kind, re := range patterns {
			if len(kinds) > 0 && !containsKind(kinds, kind) {
				continue
			}

			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				name := matches[1]

				if query != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(query)) {
					continue
				}

				symbols = append(symbols, Symbol{
					Name:       name,
					Kind:       kind,
					File:       filePath,
					Line:       lineNum + 1,
					Signature:  strings.TrimSpace(line),
					IsExported: isExported(name, lang),
				})

				if len(symbols) >= maxResults {
					return symbols
				}
			}
		}
	}

	return symbols
}

func isExported(name, lang string) bool {
	if name == "" {
		return false
	}
	switch lang {
	case "go":
		return name[0] >= 'A' && name[0] <= 'Z'
	default:
		return !strings.HasPrefix(name, "_")
	}
}

func getLanguagePatterns(lang string) map[string]*regexp.Regexp {
	switch lang {
	case "go":
		return map[string]*regexp.Regexp{
			"function":  regexp.MustCompile(`^\s*func\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`),
			"method":    regexp.MustCompile(`^\s*func\s+\([^)]+\)\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`),
			"type":      regexp.MustCompile(`^\s*type\s+([A-Za-z_][A-Za-z0-9_]*)\s+`),
			"interface": regexp.MustCompile(`^\s*type\s+([A-Za-z_][A-Za-z0-9_]*)\s+interface\s*\{`),
			"struct":    regexp.MustCompile(`^\s*type\s+([A-Za-z_][A-Za-z0-9_]*)\s+struct\s*\{`),
			"const":     regexp.MustCompile(`^\s*const\s+([A-Za-z_][A-Za-z0-9_]*)\s*`),
			"var":       regexp.MustCompile(`^\s*var\s+([A-Za-z_][A-Za-z0-9_]*)\s+`),
		}
	case "typescript", "javascript":
		return map[string]*regexp.Regexp{
			"function":  regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?function\s+([A-Za-z_$][A-Za-z0-9_$]*)`),
			"class":     regexp.MustCompile(`^\s*(?:export\s+)?class\s+([A-Za-z_$][A-Za-z0-9_$]*)`),
			"interface": regexp.MustCompile(`^\s*(?:export\s+)?interface\s+([A-Za-z_$][A-Za-z0-9_$]*)`),
			"type":      regexp.MustCompile(`^\s*(?:export\s+)?type\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=`),
			"const":     regexp.MustCompile(`^\s*(?:export\s+)?const\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*[=:]`),
		}
	case "python":
		return map[string]*regexp.Regexp{
			"function": regexp.MustCompile(`^\s*def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`),
			"class":    regexp.MustCompile(`^\s*class\s+([A-Za-z_][A-Za-z0-9_]*)`),
		}
	case "rust":
		return map[string]*regexp.Regexp{
			"function": regexp.MustCompile(`^\s*(?:pub\s+)?fn\s+([A-Za-z_][A-Za-z0-9_]*)`),
			"struct":   regexp.MustCompile(`^\s*(?:pub\s+)?struct\s+([A-Za-z_][A-Za-z0-9_]*)`),
			"enum":     regexp.MustCompile(`^\s*(?:pub\s+)?enum\s+([A-Za-z_][A-Za-z0-9_]*)`),
			"trait":    regexp.MustCompile(`^\s*(?:pub\s+)?trait\s+([A-Za-z_][A-Za-z0-9_]*)`),
		}
	case "java":
		return map[string]*regexp.Regexp{
			"class":     regexp.MustCompile(`^\s*(?:public\s+)?(?:abstract\s+)?class\s+([A-Za-z_][A-Za-z0-9_]*)`),
			"interface": regexp.MustCompile(`^\s*(?:public\s+)?interface\s+([A-Za-z_][A-Za-z0-9_]*)`),
		}
	default:
		return nil
	}
}
