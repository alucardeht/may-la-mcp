package search

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FindRequest struct {
	Pattern   string `json:"pattern"`
	Path      string `json:"path"`
	Type      string `json:"type,omitempty"`
	MaxDepth  int    `json:"max_depth,omitempty"`
	MaxResults int    `json:"max_results,omitempty"`
}

type FileInfo struct {
	Path     string    `json:"path"`
	Type     string    `json:"type"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
}

type FindResponse struct {
	Files  []FileInfo `json:"files"`
	Count  int        `json:"count"`
	Path   string     `json:"path"`
	Total  int64      `json:"total_size"`
}

type FindTool struct{}

func (t *FindTool) Name() string {
	return "find"
}

func (t *FindTool) Description() string {
	return "Find files by name pattern with glob matching"
}

func (t *FindTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"pattern": {
				"type": "string",
				"description": "Glob pattern to match (e.g., *.go, src/*/index.js)"
			},
			"path": {
				"type": "string",
				"description": "Root path to search in"
			},
			"type": {
				"type": "string",
				"description": "Type filter",
				"enum": ["file", "dir", "all"]
			},
			"max_depth": {
				"type": "integer",
				"description": "Max depth (0=unlimited)"
			},
			"max_results": {
				"type": "integer",
				"description": "Maximum number of results (default: 1000)"
			}
		},
		"required": ["pattern", "path"]
	}`)
}

func (t *FindTool) Execute(input json.RawMessage) (interface{}, error) {
	var req FindRequest
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
	if req.Type == "" {
		req.Type = "all"
	}

	files := []FileInfo{}
	totalSize := int64(0)

	err := filepath.WalkDir(req.Path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if req.MaxDepth > 0 {
			depth := strings.Count(strings.TrimPrefix(path, req.Path), string(filepath.Separator))
			if depth > req.MaxDepth {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if len(files) >= req.MaxResults {
			return filepath.SkipDir
		}

		relPath, err := filepath.Rel(req.Path, path)
		if err != nil {
			return nil
		}

		if matchesPattern(relPath, req.Pattern) {
			if shouldInclude(d, req.Type) {
				info, err := d.Info()
				if err != nil {
					return nil
				}

				fileType := "file"
				if d.IsDir() {
					fileType = "dir"
				}

				files = append(files, FileInfo{
					Path:     path,
					Type:     fileType,
					Size:     info.Size(),
					Modified: info.ModTime(),
				})
				totalSize += info.Size()

				if len(files) >= req.MaxResults {
					return filepath.SkipDir
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk error: %w", err)
	}

	return &FindResponse{
		Files:  files,
		Count:  len(files),
		Path:   req.Path,
		Total:  totalSize,
	}, nil
}

func matchesPattern(name string, pattern string) bool {
	matched, err := filepath.Match(pattern, filepath.Base(name))
	if err != nil {
		return false
	}
	if matched {
		return true
	}

	matched, err = filepath.Match(pattern, name)
	return err == nil && matched
}

func shouldInclude(d os.DirEntry, typeFilter string) bool {
	switch typeFilter {
	case "file":
		return !d.IsDir()
	case "dir":
		return d.IsDir()
	case "all":
		return true
	default:
		return true
	}
}
