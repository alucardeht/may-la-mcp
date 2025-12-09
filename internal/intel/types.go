package intel

const (
	ContentTypeCode   = "CODE"
	ContentTypeText   = "TEXT"
	ContentTypeList   = "LIST"
	ContentTypeJSON   = "JSON"
	ContentTypeGeneric = "GENERIC"
)

type ProcessingConfig struct {
	EnableSummarization  bool
	EnableTruncation     bool
	EnableRanking        bool
	EnableFormatting     bool
	EnablePatternDetection bool
	MaxContentLength     int
	ContextWindow        int
}

type IntelligenceCapability struct {
	Name        string
	Description string
	Enabled     bool
}

type IntelligenceStats struct {
	TotalProcessed     int
	AverageProcessTime float64
	PatternsDetected   int
	RankingsApplied    int
	FormattingsApplied int
}

type AnalysisResult struct {
	ContentType  string
	Patterns     []CodePattern
	Complexity   ComplexityMetrics
	Suggestions  []string
	Metrics      map[string]interface{}
}
