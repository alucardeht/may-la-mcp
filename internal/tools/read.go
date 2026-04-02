package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/alucardeht/may-la-mcp/internal/encoding"
	"github.com/alucardeht/may-la-mcp/internal/lang"
)

type ReadTool struct{}

func (t *ReadTool) Name() string { return "read" }

func (t *ReadTool) Description() string {
	return "Smart file reading with optional structural summary and batch support. Supports line ranges and symbol extraction."
}

func (t *ReadTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "File path, or comma-separated paths for batch reading"
			},
			"lines": {
				"type": "string",
				"description": "Line range: '10-50', '100-' (from line 100), '-20' (last 20 lines)"
			},
			"symbol": {
				"type": "string",
				"description": "Read a specific symbol by name"
			},
			"summary": {
				"type": "boolean",
				"description": "Prepend structural summary (auto-enabled for files >100 lines)"
			}
		},
		"required": ["path"]
	}`)
}

func (t *ReadTool) Annotations() map[string]bool {
	return map[string]bool{
		"readOnlyHint":    true,
		"destructiveHint": false,
		"idempotentHint":  true,
		"openWorldHint":   false,
	}
}

func (t *ReadTool) Execute(ctx context.Context, input json.RawMessage) (interface{}, error) {
	var params struct {
		Path    string `json:"path"`
		Lines   string `json:"lines"`
		Symbol  string `json:"symbol"`
		Summary *bool  `json:"summary"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}
	if params.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	paths := splitPaths(params.Path)

	if len(paths) > 1 {
		return readBatch(ctx, paths)
	}

	return readSingle(ctx, paths[0], params.Lines, params.Symbol, params.Summary)
}

func splitPaths(raw string) []string {
	parts := strings.Split(raw, ",")
	var result []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func readSingle(ctx context.Context, path, lineRange, symbol string, summaryFlag *bool) (string, error) {
	content, err := encoding.ReadFileAsUTF8(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", path)
		}
		if os.IsPermission(err) {
			return "", fmt.Errorf("permission denied: %s", path)
		}
		return "", fmt.Errorf("cannot read file: %w", err)
	}

	if isBinaryContent(content) {
		return fmt.Sprintf("# %s\n\n(binary file, skipped)\n", path), nil
	}

	lines := strings.Split(content, "\n")
	langID := lang.LanguageID(path)

	if symbol != "" {
		return readSymbol(path, content, lines, symbol, langID)
	}

	autoSummary := len(lines) > 100
	showSummary := autoSummary
	if summaryFlag != nil {
		showSummary = *summaryFlag
	}

	start, end := parseLineRange(lineRange, len(lines))
	displayLines := lines[start-1 : end]

	cwd, _ := os.Getwd()
	displayPath := toRelativePath(path, cwd)

	var sb strings.Builder

	if showSummary {
		symbols := lang.ExtractSymbols(content, path)
		fmt.Fprintf(&sb, "# %s (%d lines", displayPath, len(lines))
		if langID != "" {
			fmt.Fprintf(&sb, ", %s", langID)
		}
		sb.WriteString(")\n\n")

		if len(symbols) > 0 {
			sb.WriteString("## Structure\n")
			for _, sym := range symbols {
				sig := sym.Signature
				if sig == "" {
					sig = sym.Name
				}
				fmt.Fprintf(&sb, "- %-44s [line %d]\n", sig, sym.Line)
			}
			sb.WriteString("\n")
		}
		sb.WriteString("## Content\n")
	} else {
		fmt.Fprintf(&sb, "# %s (%d lines", displayPath, len(lines))
		if langID != "" {
			fmt.Fprintf(&sb, ", %s", langID)
		}
		sb.WriteString(")\n\n")
	}

	lineNumWidth := len(strconv.Itoa(end))
	for i, line := range displayLines {
		lineNum := start + i
		fmt.Fprintf(&sb, "%*d | %s\n", lineNumWidth, lineNum, line)
	}

	return sb.String(), nil
}

func readSymbol(path, content string, lines []string, symbolName, langID string) (string, error) {
	symbols := lang.ExtractSymbols(content, path)

	var targetLine int
	for _, sym := range symbols {
		if lang.MatchesQuery(sym.Name, symbolName) || strings.EqualFold(sym.Name, symbolName) {
			targetLine = sym.Line
			break
		}
	}

	if targetLine == 0 {
		return "", fmt.Errorf("symbol %q not found in %s", symbolName, path)
	}

	startLine, endLine := lang.FindFunctionBounds(lines, targetLine)
	if startLine == 0 {
		startLine = targetLine
		endLine = targetLine
	}

	cwd, _ := os.Getwd()
	displayPath := toRelativePath(path, cwd)

	var sb strings.Builder
	fmt.Fprintf(&sb, "# %s :: %s (lines %d-%d)\n\n", displayPath, symbolName, startLine, endLine)

	lineNumWidth := len(strconv.Itoa(endLine))
	for i := startLine - 1; i < endLine && i < len(lines); i++ {
		fmt.Fprintf(&sb, "%*d | %s\n", lineNumWidth, i+1, lines[i])
	}

	return sb.String(), nil
}

func readBatch(ctx context.Context, paths []string) (string, error) {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Batch Read (%d files)\n\n", len(paths))

	cwd, _ := os.Getwd()

	for _, path := range paths {
		if ctx.Err() != nil {
			return sb.String(), ctx.Err()
		}

		displayPath := toRelativePath(path, cwd)
		content, err := encoding.ReadFileAsUTF8(path)
		if err != nil {
			fmt.Fprintf(&sb, "--- %s (error: %v) ---\n\n", displayPath, err)
			continue
		}

		if isBinaryContent(content) {
			fmt.Fprintf(&sb, "--- %s (binary, skipped) ---\n\n", displayPath)
			continue
		}

		lines := strings.Split(content, "\n")
		fmt.Fprintf(&sb, "--- %s (%d lines) ---\n", displayPath, len(lines))

		lineNumWidth := len(strconv.Itoa(len(lines)))
		for i, line := range lines {
			fmt.Fprintf(&sb, "%*d | %s\n", lineNumWidth, i+1, line)
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func parseLineRange(rangeStr string, totalLines int) (int, int) {
	if rangeStr == "" {
		return 1, totalLines
	}

	if strings.HasPrefix(rangeStr, "-") {
		n, err := strconv.Atoi(rangeStr[1:])
		if err != nil || n <= 0 {
			return 1, totalLines
		}
		start := totalLines - n + 1
		if start < 1 {
			start = 1
		}
		return start, totalLines
	}

	if strings.HasSuffix(rangeStr, "-") {
		n, err := strconv.Atoi(rangeStr[:len(rangeStr)-1])
		if err != nil || n <= 0 {
			return 1, totalLines
		}
		if n > totalLines {
			n = totalLines
		}
		return n, totalLines
	}

	parts := strings.SplitN(rangeStr, "-", 2)
	if len(parts) == 2 {
		start, err1 := strconv.Atoi(parts[0])
		end, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			return 1, totalLines
		}
		if start < 1 {
			start = 1
		}
		if end > totalLines {
			end = totalLines
		}
		if start > end {
			start = end
		}
		return start, end
	}

	n, err := strconv.Atoi(rangeStr)
	if err != nil {
		return 1, totalLines
	}
	if n < 1 {
		n = 1
	}
	if n > totalLines {
		n = totalLines
	}
	return n, n
}

func isBinaryContent(content string) bool {
	sample := content
	if len(sample) > 512 {
		sample = sample[:512]
	}
	nullCount := strings.Count(sample, "\x00")
	return nullCount > 2 || (len(sample) > 0 && float64(nullCount)/float64(len(sample)) > 0.01)
}

func toRelativePath(path, base string) string {
	if base == "" {
		return path
	}
	rel, err := filepath.Rel(base, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return path
	}
	return rel
}
