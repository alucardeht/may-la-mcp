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

	"github.com/alucardeht/may-la-mcp/internal/index"
	"github.com/alucardeht/may-la-mcp/internal/router"
	"github.com/alucardeht/may-la-mcp/internal/tools"
	"github.com/alucardeht/may-la-mcp/internal/types"
)

type SymbolsRequest struct {
	Path       string   `json:"path"`
	Kinds      []string `json:"kinds,omitempty"`
	Query      string   `json:"query,omitempty"`
	MaxResults int      `json:"max_results,omitempty"`
}

type SymbolsResponse struct {
	Symbols []types.Symbol `json:"symbols"`
	Count   int            `json:"count"`
}

type SymbolsTool struct {
	router *router.Router
}

func NewSymbolsTool(r *router.Router) *SymbolsTool {
	return &SymbolsTool{router: r}
}

func (t *SymbolsTool) Name() string {
	return "symbols"
}

func (t *SymbolsTool) Description() string {
	return "Extract symbols from code files (functions, classes, methods, etc)"
}

func (t *SymbolsTool) Title() string {
	return "Extract Code Symbols"
}

func (t *SymbolsTool) Annotations() map[string]bool {
	return tools.ReadOnlyAnnotations()
}

func (t *SymbolsTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "File or directory path to extract symbols from"
			},
			"kinds": {
				"type": "array",
				"items": {
					"type": "string",
					"enum": ["function", "class", "method", "variable", "interface", "type", "const"]
				}
			},
			"query": {
				"type": "string",
				"description": "Filter symbols by name pattern"
			},
			"max_results": {
				"type": "integer",
				"description": "Maximum number of results (default: 500)"
			}
		},
		"required": ["path"]
	}`)
}

func (t *SymbolsTool) Execute(input json.RawMessage) (interface{}, error) {
	var req SymbolsRequest
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if req.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	if req.MaxResults == 0 {
		req.MaxResults = 500
	}

	ctx := context.Background()

	opts := router.QueryOptions{
		MaxResults:   req.MaxResults,
		AllowFallback: true,
	}

	if t.router != nil {
		result, err := t.router.QuerySymbols(ctx, req.Path, req.Query, req.Kinds, opts)
		if err != nil {
			return nil, fmt.Errorf("query symbols: %w", err)
		}

		symbols := make([]types.Symbol, len(result.Items))
		for i, sym := range result.Items {
			symbols[i] = types.Symbol{
				Name:      sym.Name,
				Kind:      sym.Kind,
				File:      sym.File,
				Line:      sym.Line,
				Signature: sym.Signature,
			}
		}

		return &SymbolsResponse{
			Symbols: symbols,
			Count:   len(symbols),
		}, nil
	}

	return t.executeRegex(ctx, req.Path, req.Query, req.Kinds, req.MaxResults)
}

func (t *SymbolsTool) executeRegex(ctx context.Context, path, query string, kinds []string, maxResults int) (interface{}, error) {
	kindMap := make(map[string]bool)
	if len(kinds) == 0 {
		for _, k := range []string{"function", "class", "method", "variable", "interface", "type", "const"} {
			kindMap[k] = true
		}
	} else {
		for _, k := range kinds {
			kindMap[k] = true
		}
	}

	symbols := []types.Symbol{}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("path not found: %w", err)
	}

	if info.IsDir() {
		err = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() && isSourceFile(p) {
				fileSymbols := extractSymbols(p, kindMap, query)
				symbols = append(symbols, fileSymbols...)
				if len(symbols) >= maxResults {
					return filepath.SkipDir
				}
			}
			return nil
		})
	} else {
		if isSourceFile(path) {
			symbols = extractSymbols(path, kindMap, query)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("walk error: %w", err)
	}

	if len(symbols) > maxResults {
		symbols = symbols[:maxResults]
	}

	return &SymbolsResponse{
		Symbols: symbols,
		Count:   len(symbols),
	}, nil
}

func isSourceFile(path string) bool {
	ext := filepath.Ext(path)
	sourceExts := map[string]bool{
		".go": true, ".js": true, ".ts": true, ".tsx": true, ".jsx": true,
		".py": true, ".rb": true, ".java": true, ".cpp": true, ".c": true,
		".h": true, ".cs": true, ".php": true, ".swift": true, ".kt": true,
	}
	return sourceExts[ext]
}

func extractSymbols(filePath string, kindMap map[string]bool, query string) []types.Symbol {
	content, _, err := index.ReadFileAsUTF8(filePath)
	if err != nil {
		return nil
	}

	ext := filepath.Ext(filePath)
	symbols := []types.Symbol{}

	switch ext {
	case ".go":
		symbols = extractGoSymbols(content, filePath, kindMap, query)
	case ".js", ".ts", ".tsx", ".jsx":
		symbols = extractJSSymbols(content, filePath, kindMap, query)
	case ".py":
		symbols = extractPythonSymbols(content, filePath, kindMap, query)
	case ".java":
		symbols = extractJavaSymbols(content, filePath, kindMap, query)
	}

	return symbols
}

func extractGoSymbols(content string, filePath string, kindMap map[string]bool, query string) []types.Symbol {
	symbols := []types.Symbol{}
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0

	funcRe := regexp.MustCompile(`^\s*func\s+\(([^)]*)\)\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`)
	funcRe2 := regexp.MustCompile(`^\s*func\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`)
	typeRe := regexp.MustCompile(`^\s*type\s+([a-zA-Z_][a-zA-Z0-9_]*)\s+(struct|interface)`)
	constRe := regexp.MustCompile(`^\s*const\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*=`)
	varRe := regexp.MustCompile(`^\s*var\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*=`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if match := funcRe.FindStringSubmatch(line); len(match) > 2 && kindMap["method"] {
			name := match[2]
			if matchesQuery(name, query) {
				symbols = append(symbols, types.Symbol{
					Name:      name,
					Kind:      "method",
					File:      filePath,
					Line:      lineNum,
					Signature: strings.TrimSpace(line),
				})
			}
		} else if match := funcRe2.FindStringSubmatch(line); len(match) > 1 && kindMap["function"] {
			name := match[1]
			if matchesQuery(name, query) {
				symbols = append(symbols, types.Symbol{
					Name:      name,
					Kind:      "function",
					File:      filePath,
					Line:      lineNum,
					Signature: strings.TrimSpace(line),
				})
			}
		}

		if match := typeRe.FindStringSubmatch(line); len(match) > 1 && kindMap["type"] {
			name := match[1]
			if matchesQuery(name, query) {
				kind := "type"
				if len(match) > 2 && match[2] == "interface" {
					kind = "interface"
				}
				symbols = append(symbols, types.Symbol{
					Name:      name,
					Kind:      kind,
					File:      filePath,
					Line:      lineNum,
					Signature: strings.TrimSpace(line),
				})
			}
		}

		if match := constRe.FindStringSubmatch(line); len(match) > 1 && kindMap["const"] {
			name := match[1]
			if matchesQuery(name, query) {
				symbols = append(symbols, types.Symbol{
					Name:      name,
					Kind:      "const",
					File:      filePath,
					Line:      lineNum,
					Signature: strings.TrimSpace(line),
				})
			}
		}

		if match := varRe.FindStringSubmatch(line); len(match) > 1 && kindMap["variable"] {
			name := match[1]
			if matchesQuery(name, query) {
				symbols = append(symbols, types.Symbol{
					Name:      name,
					Kind:      "variable",
					File:      filePath,
					Line:      lineNum,
					Signature: strings.TrimSpace(line),
				})
			}
		}
	}

	return symbols
}

func extractJSSymbols(content string, filePath string, kindMap map[string]bool, query string) []types.Symbol {
	symbols := []types.Symbol{}
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0

	funcRe := regexp.MustCompile(`(?:^|\s)(function|const|let|var)\s+([a-zA-Z_$][a-zA-Z0-9_$]*)\s*(?:=.*=>|\()`)
	classRe := regexp.MustCompile(`^\s*(?:export\s+)?class\s+([a-zA-Z_$][a-zA-Z0-9_$]*)`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if match := classRe.FindStringSubmatch(line); len(match) > 1 && kindMap["class"] {
			name := match[1]
			if matchesQuery(name, query) {
				symbols = append(symbols, types.Symbol{
					Name:      name,
					Kind:      "class",
					File:      filePath,
					Line:      lineNum,
					Signature: strings.TrimSpace(line),
				})
			}
		}

		if match := funcRe.FindStringSubmatch(line); len(match) > 2 {
			kind := match[1]
			name := match[2]
			symbolKind := "function"
			if (kind == "const" || kind == "let" || kind == "var") && !kindMap["variable"] {
				continue
			}
			if matchesQuery(name, query) {
				symbols = append(symbols, types.Symbol{
					Name:      name,
					Kind:      symbolKind,
					File:      filePath,
					Line:      lineNum,
					Signature: strings.TrimSpace(line),
				})
			}
		}
	}

	return symbols
}

func extractPythonSymbols(content string, filePath string, kindMap map[string]bool, query string) []types.Symbol {
	symbols := []types.Symbol{}
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0

	funcRe := regexp.MustCompile(`^\s*def\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`)
	classRe := regexp.MustCompile(`^\s*class\s+([a-zA-Z_][a-zA-Z0-9_]*)`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if match := classRe.FindStringSubmatch(line); len(match) > 1 && kindMap["class"] {
			name := match[1]
			if matchesQuery(name, query) {
				symbols = append(symbols, types.Symbol{
					Name:      name,
					Kind:      "class",
					File:      filePath,
					Line:      lineNum,
					Signature: strings.TrimSpace(line),
				})
			}
		}

		if match := funcRe.FindStringSubmatch(line); len(match) > 1 {
			name := match[1]
			kind := "function"
			if strings.HasPrefix(strings.TrimSpace(line), "\t") || strings.HasPrefix(strings.TrimSpace(line), "    ") {
				if kindMap["method"] {
					kind = "method"
				} else {
					continue
				}
			} else if !kindMap["function"] {
				continue
			}

			if matchesQuery(name, query) {
				symbols = append(symbols, types.Symbol{
					Name:      name,
					Kind:      kind,
					File:      filePath,
					Line:      lineNum,
					Signature: strings.TrimSpace(line),
				})
			}
		}
	}

	return symbols
}

func extractJavaSymbols(content string, filePath string, kindMap map[string]bool, query string) []types.Symbol {
	symbols := []types.Symbol{}
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0

	classRe := regexp.MustCompile(`\b(class|interface)\s+([a-zA-Z_][a-zA-Z0-9_]*)`)
	methodRe := regexp.MustCompile(`\b(?:public|private|protected)?\s*\w+\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if match := classRe.FindStringSubmatch(line); len(match) > 2 {
			name := match[2]
			kind := "class"
			if match[1] == "interface" && kindMap["interface"] {
				kind = "interface"
			} else if !kindMap["class"] {
				continue
			}

			if matchesQuery(name, query) {
				symbols = append(symbols, types.Symbol{
					Name:      name,
					Kind:      kind,
					File:      filePath,
					Line:      lineNum,
					Signature: strings.TrimSpace(line),
				})
			}
		}

		if match := methodRe.FindStringSubmatch(line); len(match) > 1 && kindMap["method"] {
			name := match[1]
			if matchesQuery(name, query) {
				symbols = append(symbols, types.Symbol{
					Name:      name,
					Kind:      "method",
					File:      filePath,
					Line:      lineNum,
					Signature: strings.TrimSpace(line),
				})
			}
		}
	}

	return symbols
}

func matchesQuery(name string, query string) bool {
	if query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(name), strings.ToLower(query))
}
