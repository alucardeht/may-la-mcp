package intel

import (
	"sync"
	"time"
)

type Config struct {
	EnableSummarization    bool
	EnableTruncation       bool
	EnableRanking          bool
	EnableFormatting       bool
	EnablePatternAnalysis  bool
	EnableComplexityAnalysis bool
	MaxContentLength       int
	DefaultTruncateMode    TruncateMode
	DefaultResponseMode    ResponseMode
	ContextRadius          int
	CacheResults           bool
}

var DefaultConfig = Config{
	EnableSummarization:     true,
	EnableTruncation:        true,
	EnableRanking:           false,
	EnableFormatting:        true,
	EnablePatternAnalysis:   true,
	EnableComplexityAnalysis: true,
	MaxContentLength:        3000,
	DefaultTruncateMode:     TruncateModeSmart,
	DefaultResponseMode:     ResponseModeCompact,
	ContextRadius:           5,
	CacheResults:            false,
}

type Intelligence struct {
	config Config
	stats  IntelligenceStats
	mu     sync.RWMutex
	cache  map[string]IntelligentResponse
}

func New(config Config) *Intelligence {
	return &Intelligence{
		config: config,
		stats:  IntelligenceStats{},
		cache:  make(map[string]IntelligentResponse),
	}
}

func NewDefault() *Intelligence {
	return New(DefaultConfig)
}

func (i *Intelligence) ProcessContent(content string, opts ...ProcessOption) IntelligentResponse {
	start := time.Now()
	options := &processOptions{
		ResponseConfig: DefaultResponseConfig,
	}

	for _, opt := range opts {
		opt(options)
	}

	summary := ""
	if i.config.EnableSummarization {
		summary = Summarize(content, i.config.MaxContentLength/2)
	}

	processedContent := content
	if i.config.EnableTruncation {
		processedContent = Truncate(content, i.config.MaxContentLength, i.config.DefaultTruncateMode)
	}

	response := IntelligentResponse{
		Data:        processedContent,
		Summary:     summary,
		Metadata:    make(map[string]interface{}),
		ProcessedAt: time.Now(),
	}

	if i.config.EnableFormatting {
		formatter := NewFormatterBuilder().
			WithMode(i.config.DefaultResponseMode).
			WithMaxLength(i.config.MaxContentLength).
			Build()

		formatted := formatter.Format(processedContent, response.Metadata)
		response.Data = formatted.Content
		response.Indicators = formatted.Indicators

		for k, v := range formatted.Metadata {
			response.Metadata[k] = v
		}
	}

	response.Metadata["processing_time_ms"] = time.Since(start).Milliseconds()

	i.recordStats(time.Since(start))

	return response
}

func (i *Intelligence) AnalyzeCode(code string) AnalysisResult {
	result := AnalysisResult{
		ContentType: detectContentType(code),
		Metrics:     make(map[string]interface{}),
	}

	if i.config.EnablePatternAnalysis {
		result.Patterns = DetectPatterns(code)
		result.Metrics["pattern_count"] = len(result.Patterns)
	}

	if i.config.EnableComplexityAnalysis {
		result.Complexity = AnalyzeComplexity(code)
		result.Metrics["complexity_level"] = result.Complexity.Level
		result.Metrics["cyclomatic_complexity"] = result.Complexity.CyclomaticComplexity
		result.Metrics["nesting_depth"] = result.Complexity.NestingDepth

		result.Suggestions = SuggestImprovements(code)
	}

	return result
}

func (i *Intelligence) ExtractContextAround(fileContent string, lineNum int) Context {
	return ExtractContext(fileContent, lineNum, i.config.ContextRadius)
}

func (i *Intelligence) RankContent(items []Rankable, criteria RankCriteria) []Rankable {
	if !i.config.EnableRanking {
		return items
	}

	return Rank(items, criteria)
}

func (i *Intelligence) FormatResponse(content string, mode ResponseMode) string {
	formatter := NewFormatterBuilder().
		WithMode(mode).
		WithMaxLength(i.config.MaxContentLength).
		Build()

	formatted := formatter.Format(content, make(map[string]interface{}))
	return formatted.String()
}

func (i *Intelligence) GetStats() IntelligenceStats {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.stats
}

func (i *Intelligence) ResetStats() {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.stats = IntelligenceStats{}
}

func (i *Intelligence) SetConfig(config Config) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.config = config
}

func (i *Intelligence) recordStats(duration time.Duration) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.stats.TotalProcessed++

	avgTime := i.stats.AverageProcessTime
	newAvg := (avgTime*float64(i.stats.TotalProcessed-1) + duration.Seconds()) / float64(i.stats.TotalProcessed)
	i.stats.AverageProcessTime = newAvg
}

type ProcessOption func(*processOptions)

type processOptions struct {
	ResponseConfig ResponseConfig
	Metadata       map[string]interface{}
	IncludeContext bool
}

func WithResponseMode(mode ResponseMode) ProcessOption {
	return func(opts *processOptions) {
		opts.ResponseConfig.Mode = mode
	}
}

func WithMaxLength(maxLen int) ProcessOption {
	return func(opts *processOptions) {
		opts.ResponseConfig.MaxLength = maxLen
	}
}

func WithMetadata(metadata map[string]interface{}) ProcessOption {
	return func(opts *processOptions) {
		opts.Metadata = metadata
	}
}

func WithContext(include bool) ProcessOption {
	return func(opts *processOptions) {
		opts.IncludeContext = include
	}
}

func Pipeline(content string, steps ...ProcessStep) IntelligentResponse {
	response := IntelligentResponse{
		Data:     content,
		Metadata: make(map[string]interface{}),
	}

	for _, step := range steps {
		response = step(response)
	}

	return response
}

type ProcessStep func(IntelligentResponse) IntelligentResponse

func SummarizeStep(maxLen int) ProcessStep {
	return func(ir IntelligentResponse) IntelligentResponse {
		ir.Summary = Summarize(ir.Data, maxLen)
		return ir
	}
}

func TruncateStep(maxLen int, mode TruncateMode) ProcessStep {
	return func(ir IntelligentResponse) IntelligentResponse {
		ir.Data = Truncate(ir.Data, maxLen, mode)
		return ir
	}
}

func AnalyzeStep() ProcessStep {
	return func(ir IntelligentResponse) IntelligentResponse {
		analysis := AnalyzeCode(ir.Data)
		ir.Metadata["patterns"] = analysis.Patterns
		ir.Metadata["complexity"] = analysis.Complexity
		return ir
	}
}

func FormatStep(mode ResponseMode, maxLen int) ProcessStep {
	return func(ir IntelligentResponse) IntelligentResponse {
		formatter := NewFormatterBuilder().
			WithMode(mode).
			WithMaxLength(maxLen).
			Build()

		formatted := formatter.Format(ir.Data, ir.Metadata)
		ir.Data = formatted.Content
		ir.Indicators = formatted.Indicators

		for k, v := range formatted.Metadata {
			ir.Metadata[k] = v
		}

		return ir
	}
}

func AnalyzeCode(code string) AnalysisResult {
	return AnalysisResult{
		ContentType: detectContentType(code),
		Patterns:    DetectPatterns(code),
		Complexity:  AnalyzeComplexity(code),
		Suggestions: SuggestImprovements(code),
		Metrics:     make(map[string]interface{}),
	}
}

type IntelligenceBuilder struct {
	config Config
}

func NewBuilder() *IntelligenceBuilder {
	return &IntelligenceBuilder{
		config: DefaultConfig,
	}
}

func (ib *IntelligenceBuilder) WithSummarization(enable bool) *IntelligenceBuilder {
	ib.config.EnableSummarization = enable
	return ib
}

func (ib *IntelligenceBuilder) WithTruncation(enable bool) *IntelligenceBuilder {
	ib.config.EnableTruncation = enable
	return ib
}

func (ib *IntelligenceBuilder) WithRanking(enable bool) *IntelligenceBuilder {
	ib.config.EnableRanking = enable
	return ib
}

func (ib *IntelligenceBuilder) WithFormatting(enable bool) *IntelligenceBuilder {
	ib.config.EnableFormatting = enable
	return ib
}

func (ib *IntelligenceBuilder) WithPatternAnalysis(enable bool) *IntelligenceBuilder {
	ib.config.EnablePatternAnalysis = enable
	return ib
}

func (ib *IntelligenceBuilder) WithComplexityAnalysis(enable bool) *IntelligenceBuilder {
	ib.config.EnableComplexityAnalysis = enable
	return ib
}

func (ib *IntelligenceBuilder) WithMaxLength(maxLen int) *IntelligenceBuilder {
	ib.config.MaxContentLength = maxLen
	return ib
}

func (ib *IntelligenceBuilder) WithTruncateMode(mode TruncateMode) *IntelligenceBuilder {
	ib.config.DefaultTruncateMode = mode
	return ib
}

func (ib *IntelligenceBuilder) WithResponseMode(mode ResponseMode) *IntelligenceBuilder {
	ib.config.DefaultResponseMode = mode
	return ib
}

func (ib *IntelligenceBuilder) Build() *Intelligence {
	return New(ib.config)
}
