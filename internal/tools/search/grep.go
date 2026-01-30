package search

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/alucardeht/may-la-mcp/internal/tools"
)

const MaxGrepFileSize = 100 * 1024 * 1024

type SearchRequest struct {
	Pattern       string `json:"pattern"`
	Path          string `json:"path"`
	Recursive     bool   `json:"recursive,omitempty"`
	CaseSensitive bool   `json:"case_sensitive,omitempty"`
	Regex         bool   `json:"regex,omitempty"`
	ContextLines  int    `json:"context_lines,omitempty"`
	MaxResults    int    `json:"max_results,omitempty"`
}

type Match struct {
	File    string   `json:"file"`
	Line    int      `json:"line"`
	Column  int      `json:"column"`
	Content string   `json:"content"`
	Context []string `json:"context,omitempty"`
}

type SearchResponse struct {
	Matches []Match `json:"matches"`
	Count   int     `json:"count"`
	Path    string  `json:"path"`
}

type SearchTool struct{}

func (t *SearchTool) Name() string {
	return "search"
}

func (t *SearchTool) Description() string {
	return "Search for pattern in files with regex and context support"
}

func (t *SearchTool) Title() string {
	return "Search File Contents"
}

func (t *SearchTool) Annotations() map[string]bool {
	return tools.ReadOnlyAnnotations()
}

func (t *SearchTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"pattern": {
				"type": "string",
				"description": "Search pattern (regex if regex=true)"
			},
			"path": {
				"type": "string",
				"description": "Root path to search in"
			},
			"recursive": {
				"type": "boolean",
				"description": "Search subdirs"
			},
			"case_sensitive": {
				"type": "boolean",
				"description": "Case-sensitive search (default: false)"
			},
			"regex": {
				"type": "boolean",
				"description": "Treat pattern as regex (default: false)"
			},
			"context_lines": {
				"type": "integer",
				"description": "Context lines around match"
			},
			"max_results": {
				"type": "integer",
				"description": "Maximum number of results (default: 1000)"
			}
		},
		"required": ["pattern", "path"]
	}`)
}

func (t *SearchTool) Execute(ctx context.Context, input json.RawMessage) (interface{}, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	var req SearchRequest
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if req.Pattern == "" {
		return nil, fmt.Errorf("pattern is required")
	}
	if req.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	if req.MaxResults == 0 {
		req.MaxResults = 1000
	}
	if req.ContextLines < 0 {
		req.ContextLines = 0
	}

	rgOutput, err := executeRipgrep(req)
	if err == nil && rgOutput != nil {
		return rgOutput, nil
	}

	return searchWithGo(ctx, req)
}

func searchWithGo(ctx context.Context, req SearchRequest) (interface{}, error) {
	var pattern *regexp.Regexp
	var err error

	if req.Regex {
		flags := ""
		if !req.CaseSensitive {
			flags = "(?i)"
		}
		pattern, err = regexp.Compile(flags + req.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex: %w", err)
		}
	}

	matches := []Match{}
	visited := make(map[string]bool)

	err = filepath.WalkDir(req.Path, func(path string, d os.DirEntry, err error) error {
		// Check for context cancellation to respect timeouts
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err != nil {
			return nil
		}

		if d.IsDir() {
			if !req.Recursive && path != req.Path {
				return filepath.SkipDir
			}
			return nil
		}

		if visited[path] {
			return nil
		}
		visited[path] = true

		if len(matches) >= req.MaxResults {
			return filepath.SkipDir
		}

		fileMatches := searchFile(path, req, pattern)
		matches = append(matches, fileMatches...)

		if len(matches) > req.MaxResults {
			matches = matches[:req.MaxResults]
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk error: %w", err)
	}

	return &SearchResponse{
		Matches: matches,
		Count:   len(matches),
		Path:    req.Path,
	}, nil
}

func searchFile(filePath string, req SearchRequest, pattern *regexp.Regexp) []Match {
	fileInfo, err := os.Stat(filePath)
	if err == nil && fileInfo.Size() > MaxGrepFileSize {
		return nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer file.Close()

	matches := []Match{}
	scanner := bufio.NewScanner(file)
	lineNum := 0

	var lines []string
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		lines = append(lines, line)
	}

	scanner = bufio.NewScanner(file)
	file.Seek(0, 0)
	lineNum = 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		var found bool
		var column int

		if req.Regex {
			loc := pattern.FindStringIndex(line)
			if loc != nil {
				found = true
				column = loc[0] + 1
			}
		} else {
			searchStr := req.Pattern
			if !req.CaseSensitive {
				searchStr = strings.ToLower(searchStr)
				if idx := strings.Index(strings.ToLower(line), searchStr); idx >= 0 {
					found = true
					column = idx + 1
				}
			} else {
				if idx := strings.Index(line, searchStr); idx >= 0 {
					found = true
					column = idx + 1
				}
			}
		}

		if found {
			m := Match{
				File:    filePath,
				Line:    lineNum,
				Column:  column,
				Content: line,
			}

			if req.ContextLines > 0 {
				m.Context = getContextLines(lines, lineNum-1, req.ContextLines)
			}

			matches = append(matches, m)

			if len(matches) >= req.MaxResults {
				break
			}
		}
	}

	return matches
}

func getContextLines(lines []string, matchIdx int, contextLines int) []string {
	result := []string{}
	start := matchIdx - contextLines
	if start < 0 {
		start = 0
	}

	end := matchIdx + contextLines + 1
	if end > len(lines) {
		end = len(lines)
	}

	for i := start; i < end; i++ {
		result = append(result, lines[i])
	}

	return result
}
