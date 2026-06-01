package tools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/alucardeht/may-la-mcp/internal/encoding"
	"github.com/alucardeht/may-la-mcp/internal/gitignore"
	"github.com/alucardeht/may-la-mcp/internal/lang"
	"github.com/bmatcuk/doublestar/v4"
)

type SearchTool struct{}

func (t *SearchTool) Name() string { return "search" }

func (t *SearchTool) Description() string {
	return "Search by content and return complete function bodies containing matches. Supports regex, glob filtering, and context modes."
}

func (t *SearchTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"pattern": {
				"type": "string",
				"description": "Search term or regex pattern"
			},
			"path": {
				"type": "string",
				"description": "Root path to search in"
			},
			"regex": {
				"type": "boolean",
				"description": "Treat pattern as regex (default false)",
				"default": false
			},
			"include": {
				"type": "string",
				"description": "Glob pattern to filter files (e.g. '*.go', '**/*.ts')"
			},
			"context": {
				"type": "string",
				"description": "Context mode: 'lines', 'function', 'block' (default 'function')",
				"enum": ["lines", "function", "block"],
				"default": "function"
			},
			"max_results": {
				"type": "integer",
				"description": "Maximum number of results (default 20)",
				"default": 20
			}
		},
		"required": ["pattern", "path"]
	}`)
}

func (t *SearchTool) Annotations() map[string]bool {
	return map[string]bool{
		"readOnlyHint":    true,
		"destructiveHint": false,
		"idempotentHint":  true,
		"openWorldHint":   false,
	}
}

type searchMatch struct {
	filePath    string
	lineNum     int
	contextText string
	funcName    string
}

func (t *SearchTool) Execute(ctx context.Context, input json.RawMessage) (interface{}, error) {
	var params struct {
		Pattern    string `json:"pattern"`
		Path       string `json:"path"`
		Regex      bool   `json:"regex"`
		Include    string `json:"include"`
		Context    string `json:"context"`
		MaxResults int    `json:"max_results"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}
	if params.Pattern == "" {
		return nil, fmt.Errorf("pattern is required")
	}
	if params.Path == "" {
		return nil, fmt.Errorf("path is required")
	}
	if params.Context == "" {
		params.Context = "function"
	}
	if params.MaxResults <= 0 {
		params.MaxResults = 20
	}

	matches, err := collectMatches(ctx, params.Pattern, params.Path, params.Regex, params.Include, params.MaxResults)
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		return fmt.Sprintf("# Search: %q in %s\n\nNo matches found.\n", params.Pattern, params.Path), nil
	}

	matcher := gitignore.New(params.Path)
	results, err := buildSearchOutput(ctx, params.Pattern, params.Path, matches, params.Context, matcher)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func collectMatches(ctx context.Context, pattern, root string, isRegex bool, include string, maxResults int) ([]searchMatch, error) {
	rgPath, err := exec.LookPath("rg")
	if err == nil {
		return collectWithRipgrep(ctx, rgPath, pattern, root, isRegex, include, maxResults)
	}
	return collectNative(ctx, pattern, root, isRegex, include, maxResults)
}

func collectWithRipgrep(ctx context.Context, rgPath, pattern, root string, isRegex bool, include string, maxResults int) ([]searchMatch, error) {
	args := []string{"--no-heading", "--color=never", "-n", "--with-filename"}
	if !isRegex {
		args = append(args, "-F")
	}
	if include != "" {
		args = append(args, "--glob", include)
	}
	args = append(args, "--", pattern, root)

	cmd := exec.CommandContext(ctx, rgPath, args...)
	out, err := cmd.Output()
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		return collectNative(ctx, pattern, root, isRegex, include, maxResults)
	}

	var matches []searchMatch
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		if ctx.Err() != nil {
			break
		}
		if len(matches) >= maxResults {
			break
		}
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}
		filePath := parts[0]
		lineNum, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		matches = append(matches, searchMatch{
			filePath: filePath,
			lineNum:  lineNum,
		})
	}
	return matches, nil
}

func collectNative(ctx context.Context, pattern, root string, isRegex bool, include string, maxResults int) ([]searchMatch, error) {
	var re *regexp.Regexp
	var err error

	if isRegex {
		re, err = regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex: %w", err)
		}
	}

	matcher := gitignore.New(root)
	var matches []searchMatch

	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if ctx.Err() != nil {
			return filepath.SkipAll
		}
		if err != nil || info == nil {
			return nil
		}
		relPath, _ := filepath.Rel(root, path)
		relPath = filepath.ToSlash(relPath)
		if matcher.IsIgnored(relPath, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !gitignore.IsSourceFile(path) {
			return nil
		}
		if include != "" {
			matched, _ := doublestar.PathMatch(include, filepath.Base(path))
			if !matched {
				matched, _ = doublestar.PathMatch(include, relPath)
				if !matched {
					return nil
				}
			}
		}
		if len(matches) >= maxResults {
			return filepath.SkipAll
		}

		content, err := encoding.ReadFileAsUTF8(path)
		if err != nil || isBinaryContent(content) {
			return nil
		}

		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if ctx.Err() != nil {
				return filepath.SkipAll
			}
			if len(matches) >= maxResults {
				return filepath.SkipAll
			}
			var found bool
			if isRegex {
				found = re.MatchString(line)
			} else {
				found = strings.Contains(line, pattern)
			}
			if found {
				matches = append(matches, searchMatch{
					filePath: path,
					lineNum:  i + 1,
				})
			}
		}
		return nil
	})

	return matches, nil
}

func buildSearchOutput(ctx context.Context, pattern, root string, matches []searchMatch, contextMode string, matcher *gitignore.Matcher) (string, error) {
	cwd, _ := os.Getwd()

	type dedupKey struct {
		file  string
		start int
	}
	seen := make(map[dedupKey]bool)
	var enriched []searchMatch

	for _, m := range matches {
		if ctx.Err() != nil {
			break
		}

		content, err := encoding.ReadFileAsUTF8(m.filePath)
		if err != nil {
			continue
		}

		lines := strings.Split(content, "\n")
		var contextText string
		var funcName string

		switch contextMode {
		case "lines":
			contextText, funcName = buildLinesContext(lines, m.lineNum)
		case "block":
			contextText, funcName = buildBlockContext(lines, m.lineNum)
		default:
			contextText, funcName = buildFunctionContext(lines, m.lineNum)
		}

		startLine := m.lineNum
		if contextMode == "function" || contextMode == "block" {
			start, _ := lang.FindFunctionBounds(lines, m.lineNum)
			if start > 0 {
				startLine = start
			}
		}

		key := dedupKey{file: m.filePath, start: startLine}
		if seen[key] {
			continue
		}
		seen[key] = true

		m.contextText = contextText
		m.funcName = funcName
		enriched = append(enriched, m)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Search: %q in %s (%d matches)\n\n", pattern, root, len(enriched))

	for i, m := range enriched {
		displayPath := toRelativePath(m.filePath, cwd)
		if m.funcName != "" {
			fmt.Fprintf(&sb, "## Match %d: %s:%d  [in %s]\n", i+1, displayPath, m.lineNum, m.funcName)
		} else {
			fmt.Fprintf(&sb, "## Match %d: %s:%d\n", i+1, displayPath, m.lineNum)
		}
		sb.WriteString(m.contextText)
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func buildLinesContext(lines []string, lineNum int) (string, string) {
	contextRadius := 3
	start := lineNum - contextRadius - 1
	end := lineNum + contextRadius

	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}

	funcName := lang.FindEnclosingFunction(lines, lineNum)
	lineNumWidth := len(strconv.Itoa(end))

	var sb strings.Builder
	for i := start; i < end; i++ {
		fmt.Fprintf(&sb, "%*d | %s\n", lineNumWidth, i+1, lines[i])
	}
	return sb.String(), funcName
}

func buildFunctionContext(lines []string, lineNum int) (string, string) {
	startLine, endLine := lang.FindFunctionBounds(lines, lineNum)
	if startLine == 0 {
		return buildLinesContext(lines, lineNum)
	}

	funcName := lang.FindEnclosingFunction(lines, lineNum)
	lineNumWidth := len(strconv.Itoa(endLine))

	funcLines := lines[startLine-1 : endLine]
	const maxFirstLines = 20
	const maxLastLines = 10
	const maxTotalLines = 60

	var sb strings.Builder
	if len(funcLines) > maxTotalLines {
		middle := len(funcLines) - maxFirstLines - maxLastLines
		for i := 0; i < maxFirstLines; i++ {
			fmt.Fprintf(&sb, "%*d | %s\n", lineNumWidth, startLine+i, funcLines[i])
		}
		fmt.Fprintf(&sb, "       // ... (%d more lines)\n", middle)
		tailStart := len(funcLines) - maxLastLines
		for i := tailStart; i < len(funcLines); i++ {
			fmt.Fprintf(&sb, "%*d | %s\n", lineNumWidth, startLine+i, funcLines[i])
		}
	} else {
		for i, line := range funcLines {
			fmt.Fprintf(&sb, "%*d | %s\n", lineNumWidth, startLine+i, line)
		}
	}

	return sb.String(), funcName
}

func buildBlockContext(lines []string, lineNum int) (string, string) {
	if lineNum < 1 || lineNum > len(lines) {
		return buildLinesContext(lines, lineNum)
	}

	startLine := lineNum
	for i := lineNum - 1; i >= 0 && i >= lineNum-50; i-- {
		if strings.Contains(lines[i], "{") {
			startLine = i + 1
			break
		}
	}

	depth := 0
	endLine := lineNum
	for i := startLine - 1; i < len(lines); i++ {
		for _, ch := range lines[i] {
			if ch == '{' {
				depth++
			} else if ch == '}' {
				depth--
				if depth <= 0 {
					endLine = i + 1
					goto blockFound
				}
			}
		}
	}
blockFound:

	funcName := lang.FindEnclosingFunction(lines, lineNum)
	lineNumWidth := len(strconv.Itoa(endLine))

	var sb strings.Builder
	for i := startLine - 1; i < endLine && i < len(lines); i++ {
		fmt.Fprintf(&sb, "%*d | %s\n", lineNumWidth, i+1, lines[i])
	}
	return sb.String(), funcName
}
