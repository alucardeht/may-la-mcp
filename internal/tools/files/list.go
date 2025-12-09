package files

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ListRequest struct {
	Path      string `json:"path"`
	Recursive bool   `json:"recursive,omitempty"`
	Pattern   string `json:"pattern,omitempty"`
	ShowHidden bool  `json:"showHidden,omitempty"`
	SortBy    string `json:"sortBy,omitempty"`
}

type FileInfo struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Type        string    `json:"type"`
	Size        int64     `json:"size"`
	Modified    time.Time `json:"modified"`
	Permissions string    `json:"permissions"`
}

type ListResponse struct {
	Path  string     `json:"path"`
	Files []FileInfo `json:"files"`
	Count int        `json:"count"`
}

type ListTool struct{}

func (t *ListTool) Name() string {
	return "list"
}

func (t *ListTool) Description() string {
	return "List directory contents with filtering and sorting options"
}

func (t *ListTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Directory path to list (absolute path required)"
			},
			"recursive": {
				"type": "boolean",
				"description": "List recursively (default: false)"
			},
			"pattern": {
				"type": "string",
				"description": "Glob pattern to filter files (e.g., *.go)"
			},
			"showHidden": {
				"type": "boolean",
				"description": "Show hidden files/dirs starting with . (default: false)"
			},
			"sortBy": {
				"type": "string",
				"description": "Sort by: name, size, date (default: name)",
				"enum": ["name", "size", "date"]
			}
		},
		"required": ["path"]
	}`)
}

func (t *ListTool) Execute(input json.RawMessage) (interface{}, error) {
	var req ListRequest
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if req.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	stat, err := os.Stat(req.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path does not exist")
		}
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	if !stat.IsDir() {
		return nil, fmt.Errorf("path is not a directory")
	}

	var files []FileInfo

	if req.Recursive {
		err = filepath.Walk(req.Path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if path == req.Path {
				return nil
			}

			if !req.ShowHidden && strings.HasPrefix(info.Name(), ".") {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			if req.Pattern != "" {
				matched, err := filepath.Match(req.Pattern, info.Name())
				if err != nil || !matched {
					return nil
				}
			}

			itemType := "file"
			if info.IsDir() {
				itemType = "dir"
			}

			files = append(files, FileInfo{
				Name:        info.Name(),
				Path:        path,
				Type:        itemType,
				Size:        info.Size(),
				Modified:    info.ModTime(),
				Permissions: info.Mode().String(),
			})

			return nil
		})
	} else {
		entries, err := os.ReadDir(req.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %w", err)
		}

		for _, entry := range entries {
			if !req.ShowHidden && strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			if req.Pattern != "" {
				matched, err := filepath.Match(req.Pattern, entry.Name())
				if err != nil || !matched {
					continue
				}
			}

			info, err := entry.Info()
			if err != nil {
				continue
			}

			itemType := "file"
			if entry.IsDir() {
				itemType = "dir"
			}

			files = append(files, FileInfo{
				Name:        entry.Name(),
				Path:        filepath.Join(req.Path, entry.Name()),
				Type:        itemType,
				Size:        info.Size(),
				Modified:    info.ModTime(),
				Permissions: info.Mode().String(),
			})
		}
	}

	sortFiles(files, req.SortBy)

	return ListResponse{
		Path:  req.Path,
		Files: files,
		Count: len(files),
	}, nil
}

func sortFiles(files []FileInfo, sortBy string) {
	switch sortBy {
	case "size":
		sort.Slice(files, func(i, j int) bool {
			return files[i].Size < files[j].Size
		})
	case "date":
		sort.Slice(files, func(i, j int) bool {
			return files[i].Modified.Before(files[j].Modified)
		})
	default:
		sort.Slice(files, func(i, j int) bool {
			return files[i].Name < files[j].Name
		})
	}
}
