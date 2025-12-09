package intel

import (
	"bytes"
	"fmt"
	"strings"
)

type ResponseMode string

const (
	ResponseModeCompact  ResponseMode = "COMPACT"
	ResponseModeDetailed ResponseMode = "DETAILED"
	ResponseModeRaw      ResponseMode = "RAW"
)

type ResponseFormatter struct {
	Mode           ResponseMode
	MaxLength      int
	IncludeMetrics bool
	IncludeContext bool
	LineLimitHead  int
	LineLimitTail  int
}

type FormattedResponse struct {
	Mode       ResponseMode
	Title      string
	Summary    string
	Content    string
	Metadata   map[string]interface{}
	Indicators []string
}

var DefaultFormatter = ResponseFormatter{
	Mode:           ResponseModeCompact,
	MaxLength:      2000,
	IncludeMetrics: true,
	IncludeContext: false,
	LineLimitHead:  5,
	LineLimitTail:  3,
}

func (rf *ResponseFormatter) Format(content string, metadata map[string]interface{}) FormattedResponse {
	switch rf.Mode {
	case ResponseModeCompact:
		return rf.formatCompact(content, metadata)
	case ResponseModeDetailed:
		return rf.formatDetailed(content, metadata)
	case ResponseModeRaw:
		return rf.formatRaw(content, metadata)
	default:
		return rf.formatCompact(content, metadata)
	}
}

func (rf *ResponseFormatter) formatCompact(content string, metadata map[string]interface{}) FormattedResponse {
	summary := summarizeForCompact(content, rf.MaxLength/2)
	truncated := Truncate(content, rf.MaxLength, TruncateModeSmart)

	indicators := buildIndicators(content, metadata)

	return FormattedResponse{
		Mode:       ResponseModeCompact,
		Summary:    summary,
		Content:    truncated,
		Metadata:   metadata,
		Indicators: indicators,
	}
}

func (rf *ResponseFormatter) formatDetailed(content string, metadata map[string]interface{}) FormattedResponse {
	lines := strings.Split(content, "\n")

	headLines := rf.LineLimitHead
	if headLines > len(lines) {
		headLines = len(lines)
	}

	tailLines := rf.LineLimitTail
	if tailLines+headLines > len(lines) {
		tailLines = len(lines) - headLines
	}

	var buf bytes.Buffer

	for i := 0; i < headLines; i++ {
		if i > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString(fmt.Sprintf("%4d | %s", i+1, lines[i]))
	}

	if headLines+tailLines < len(lines) {
		omittedCount := len(lines) - headLines - tailLines
		buf.WriteString(fmt.Sprintf("\n     ... [%d lines omitted] ...\n", omittedCount))

		for i := 0; i < tailLines; i++ {
			lineNum := len(lines) - tailLines + i
			buf.WriteString(fmt.Sprintf("%4d | %s", lineNum+1, lines[lineNum]))
			if i < tailLines-1 {
				buf.WriteString("\n")
			}
		}
	}

	enrichedMetadata := enrichMetadata(metadata, content)

	return FormattedResponse{
		Mode:       ResponseModeDetailed,
		Content:    buf.String(),
		Metadata:   enrichedMetadata,
		Indicators: buildDetailedIndicators(content, enrichedMetadata),
	}
}

func (rf *ResponseFormatter) formatRaw(content string, metadata map[string]interface{}) FormattedResponse {
	return FormattedResponse{
		Mode:     ResponseModeRaw,
		Content:  content,
		Metadata: metadata,
	}
}

func summarizeForCompact(content string, maxLen int) string {
	lines := strings.Split(content, "\n")

	if len(lines) <= 3 {
		return Summarize(content, maxLen)
	}

	var summary strings.Builder
	summary.WriteString(lines[0])

	for i := 1; i < len(lines) && i < 3; i++ {
		if summary.Len() > maxLen {
			break
		}
		summary.WriteString("\n" + lines[i])
	}

	return summary.String()
}

func buildIndicators(content string, metadata map[string]interface{}) []string {
	var indicators []string

	lines := strings.Split(content, "\n")
	indicators = append(indicators, fmt.Sprintf("📄 Lines: %d", len(lines)))

	if count, ok := metadata["symbol_count"].(int); ok {
		indicators = append(indicators, fmt.Sprintf("📊 Symbols: %d", count))
	}

	if count, ok := metadata["match_count"].(int); ok {
		indicators = append(indicators, fmt.Sprintf("🎯 Matches: %d", count))
	}

	if duration, ok := metadata["search_time"].(string); ok {
		indicators = append(indicators, fmt.Sprintf("⏱ Time: %s", duration))
	}

	return indicators
}

func buildDetailedIndicators(content string, metadata map[string]interface{}) []string {
	var indicators []string

	lines := strings.Split(content, "\n")
	words := countWords(content)
	chars := len(content)

	indicators = append(indicators, fmt.Sprintf("Lines: %d", len(lines)))
	indicators = append(indicators, fmt.Sprintf("Words: %d", words))
	indicators = append(indicators, fmt.Sprintf("Characters: %d", chars))

	if complexity, ok := metadata["complexity"].(string); ok {
		indicators = append(indicators, fmt.Sprintf("Complexity: %s", complexity))
	}

	if refs, ok := metadata["references"].(int); ok {
		indicators = append(indicators, fmt.Sprintf("References: %d", refs))
	}

	return indicators
}

func enrichMetadata(metadata map[string]interface{}, content string) map[string]interface{} {
	enriched := make(map[string]interface{})

	for k, v := range metadata {
		enriched[k] = v
	}

	lines := strings.Split(content, "\n")
	enriched["line_count"] = len(lines)
	enriched["word_count"] = countWords(content)
	enriched["char_count"] = len(content)

	enriched["has_code"] = detectHasCode(content)
	enriched["complexity"] = analyzeComplexity(content)

	return enriched
}

func countWords(content string) int {
	return len(strings.Fields(content))
}

func detectHasCode(content string) bool {
	codeIndicators := []string{
		"func ", "function ", "class ", "interface ", "type ",
		"const ", "var ", "let ", "import ", "export ",
		"{", "}", "=>", "->", "::",
	}

	lower := strings.ToLower(content)
	for _, indicator := range codeIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}

	return false
}

func analyzeComplexity(content string) string {
	lines := strings.Split(content, "\n")
	codeLines := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isCodeLine(trimmed) {
			codeLines++
		}
	}

	ratio := float64(codeLines) / float64(len(lines))

	if ratio > 0.7 {
		if containsNesting(content) {
			return "HIGH"
		}
		return "MEDIUM"
	}

	return "LOW"
}

func containsNesting(content string) bool {
	maxDepth := 0
	currentDepth := 0

	for _, char := range content {
		switch char {
		case '{', '[', '(':
			currentDepth++
			if currentDepth > maxDepth {
				maxDepth = currentDepth
			}
		case '}', ']', ')':
			currentDepth--
		}
	}

	return maxDepth > 3
}

type FormatterBuilder struct {
	formatter ResponseFormatter
}

func NewFormatterBuilder() *FormatterBuilder {
	return &FormatterBuilder{
		formatter: DefaultFormatter,
	}
}

func (fb *FormatterBuilder) WithMode(mode ResponseMode) *FormatterBuilder {
	fb.formatter.Mode = mode
	return fb
}

func (fb *FormatterBuilder) WithMaxLength(maxLen int) *FormatterBuilder {
	fb.formatter.MaxLength = maxLen
	return fb
}

func (fb *FormatterBuilder) WithMetrics(include bool) *FormatterBuilder {
	fb.formatter.IncludeMetrics = include
	return fb
}

func (fb *FormatterBuilder) WithContext(include bool) *FormatterBuilder {
	fb.formatter.IncludeContext = include
	return fb
}

func (fb *FormatterBuilder) WithLineLimits(head, tail int) *FormatterBuilder {
	fb.formatter.LineLimitHead = head
	fb.formatter.LineLimitTail = tail
	return fb
}

func (fb *FormatterBuilder) Build() ResponseFormatter {
	return fb.formatter
}

func (fr FormattedResponse) String() string {
	var buf bytes.Buffer

	if fr.Summary != "" {
		buf.WriteString(fr.Summary)
		buf.WriteString("\n\n")
	}

	buf.WriteString(fr.Content)

	if len(fr.Indicators) > 0 {
		buf.WriteString("\n\n")
		for i, indicator := range fr.Indicators {
			if i > 0 {
				buf.WriteString(" | ")
			}
			buf.WriteString(indicator)
		}
	}

	return buf.String()
}
