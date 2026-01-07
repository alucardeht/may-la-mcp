package search

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type ReferencesRequest struct {
	Symbol     string `json:"symbol"`
	Path       string `json:"path"`
	Recursive  bool   `json:"recursive,omitempty"`
	MaxResults int    `json:"max_results,omitempty"`
}

type Reference struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Context string `json:"context"`
	Kind    string `json:"kind"`
}

type ReferencesResponse struct {
	References []Reference `json:"references"`
	Count      int         `json:"count"`
	Symbol     string      `json:"symbol"`
}

type ReferencesTool struct{}

func (t *ReferencesTool) Name() string {
	return "references"
}

func (t *ReferencesTool) Description() string {
	return "Find references to a symbol with word boundary matching"
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

	if !req.Recursive {
		req.Recursive = true
	}

	references := []Reference{}
	wordBoundaryPattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(req.Symbol) + `\b`)

	err := filepath.WalkDir(req.Path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			if !req.Recursive && path != req.Path {
				return filepath.SkipDir
			}
			return nil
		}

		if len(references) >= req.MaxResults {
			return filepath.SkipDir
		}

		if isSourceFile(path) {
			fileRefs := findReferencesInFile(path, req.Symbol, wordBoundaryPattern)
			references = append(references, fileRefs...)

			if len(references) > req.MaxResults {
				references = references[:req.MaxResults]
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk error: %w", err)
	}

	return &ReferencesResponse{
		References: references,
		Count:      len(references),
		Symbol:     req.Symbol,
	}, nil
}

func findReferencesInFile(filePath string, symbol string, pattern *regexp.Regexp) []Reference {
	file, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer file.Close()

	references := []Reference{}
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		locs := pattern.FindAllStringIndex(line, -1)
		for _, loc := range locs {
			column := loc[0] + 1
			kind := determineReferenceKind(line, loc[0], symbol)

			references = append(references, Reference{
				File:    filePath,
				Line:    lineNum,
				Column:  column,
				Context: strings.TrimSpace(line),
				Kind:    kind,
			})
		}
	}

	return references
}

func determineReferenceKind(line string, position int, symbol string) string {
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
