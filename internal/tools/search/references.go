package search

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/alucardeht/may-la-mcp/internal/router"
	"github.com/alucardeht/may-la-mcp/internal/tools"
	"github.com/alucardeht/may-la-mcp/internal/types"
)

type ReferencesRequest struct {
	Symbol     string `json:"symbol"`
	Path       string `json:"path"`
	Recursive  bool   `json:"recursive,omitempty"`
	MaxResults int    `json:"max_results,omitempty"`
}

type ReferencesResponse struct {
	References []types.Reference `json:"references"`
	Count      int               `json:"count"`
	Symbol     string            `json:"symbol"`
}

type ReferencesTool struct {
	router *router.Router
}

func NewReferencesTool(r *router.Router) *ReferencesTool {
	return &ReferencesTool{router: r}
}

func (t *ReferencesTool) Name() string {
	return "references"
}

func (t *ReferencesTool) Description() string {
	return "Find references to a symbol with word boundary matching"
}

func (t *ReferencesTool) Title() string {
	return "Find Symbol References"
}

func (t *ReferencesTool) Annotations() map[string]bool {
	return tools.ReadOnlyAnnotations()
}

func (t *ReferencesTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"symbol": {
				"type": "string",
				"description": "Symbol name to find references for"
			},
			"path": {
				"type": "string",
				"description": "Root path to search in"
			},
			"recursive": {
				"type": "boolean",
				"description": "Search subdirs"
			},
			"max_results": {
				"type": "integer",
				"description": "Maximum number of results (default: 1000)"
			}
		},
		"required": ["symbol", "path"]
	}`)
}

func (t *ReferencesTool) Execute(input json.RawMessage) (interface{}, error) {
	var req ReferencesRequest
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if req.Symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	if req.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	if req.MaxResults == 0 {
		req.MaxResults = 1000
	}

	ctx := context.Background()

	opts := router.QueryOptions{
		MaxResults:    req.MaxResults,
		AllowFallback: true,
	}

	if t.router != nil {
		result, err := t.router.QueryReferences(ctx, req.Symbol, req.Path, opts)
		if err != nil {
			return nil, fmt.Errorf("query references: %w", err)
		}

		references := make([]types.Reference, len(result.Items))
		for i, ref := range result.Items {
			references[i] = types.Reference{
				File:    ref.File,
				Line:    ref.Line,
				Column:  ref.Column,
				Context: ref.Context,
				Kind:    ref.Kind,
			}
		}

		return &ReferencesResponse{
			References: references,
			Count:      len(references),
			Symbol:     req.Symbol,
		}, nil
	}

	return t.executeRegex(ctx, req.Symbol, req.Path, req.MaxResults)
}

func (t *ReferencesTool) executeRegex(ctx context.Context, symbol, path string, maxResults int) (interface{}, error) {
	result, err := findReferencesRegex(ctx, symbol, path, maxResults)
	if err != nil {
		return nil, fmt.Errorf("find references: %w", err)
	}

	return &ReferencesResponse{
		References: result,
		Count:      len(result),
		Symbol:     symbol,
	}, nil
}

func findReferencesRegex(ctx context.Context, symbol string, searchPath string, maxResults int) ([]types.Reference, error) {
	var references []types.Reference

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

		if !isSourceFile(path) {
			return nil
		}

		fileInfo, err := os.Stat(path)
		if err != nil {
			return nil
		}
		if fileInfo.Size() > 100*1024*1024 {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(content), "\n")
		for lineNum, line := range lines {
			if pattern.MatchString(line) {
				locs := pattern.FindAllStringIndex(line, -1)
				for _, loc := range locs {
					column := loc[0] + 1
					kind := classifyReferenceKind(line, loc[0], symbol)

					references = append(references, types.Reference{
						File:    path,
						Line:    lineNum + 1,
						Column:  column,
						Context: strings.TrimSpace(line),
						Kind:    kind,
					})

					if len(references) >= maxResults {
						return filepath.SkipAll
					}
				}
			}
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return nil, err
	}

	return references, nil
}

func classifyReferenceKind(line string, position int, symbol string) string {
	beforeContext := line[:position]

	if strings.Contains(beforeContext, "//") {
		return "comment"
	}

	if strings.Contains(beforeContext, "\"") || strings.Contains(beforeContext, "'") {
		quoteCount := strings.Count(beforeContext, "\"") + strings.Count(beforeContext, "'")
		if quoteCount%2 != 0 {
			return "string"
		}
	}

	beforeTrimmed := strings.TrimSpace(beforeContext)

	if strings.HasSuffix(beforeTrimmed, "func") || strings.HasSuffix(beforeTrimmed, "type") ||
		strings.HasSuffix(beforeTrimmed, "const") || strings.HasSuffix(beforeTrimmed, "var") ||
		strings.HasSuffix(beforeTrimmed, "class") || strings.HasSuffix(beforeTrimmed, "interface") {
		return "definition"
	}

	if strings.HasSuffix(beforeTrimmed, "import") || strings.HasSuffix(beforeTrimmed, "from") ||
		strings.HasSuffix(beforeTrimmed, "require") {
		return "import"
	}

	return "usage"
}
