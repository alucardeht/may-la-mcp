package files

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/alucardeht/may-la-mcp/internal/tools"
)

type InfoRequest struct {
	Path string `json:"path"`
}

type FileSystemInfo struct {
	Path        string    `json:"path"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Size        int64     `json:"size"`
	Permissions string    `json:"permissions"`
	Mode        uint32    `json:"mode"`
	Owner       string    `json:"owner"`
	Created     time.Time `json:"created"`
	Modified    time.Time `json:"modified"`
	Accessed    time.Time `json:"accessed"`
	IsSymlink   bool      `json:"isSymlink"`
	FileCount   int       `json:"fileCount,omitempty"`
	TotalSize   int64     `json:"totalSize,omitempty"`
}

type InfoTool struct{}

func (t *InfoTool) Name() string {
	return "info"
}

func (t *InfoTool) Description() string {
	return "Get detailed information about file or directory"
}

func (t *InfoTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Path to get info about (absolute path required)"
			}
		},
		"required": ["path"]
	}`)
}

func (t *InfoTool) Execute(input json.RawMessage) (interface{}, error) {
	var req InfoRequest
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if req.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	stat, err := os.Lstat(req.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path does not exist")
		}
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	itemType := "file"
	if stat.IsDir() {
		itemType = "dir"
	} else if stat.Mode()&os.ModeSymlink != 0 {
		itemType = "symlink"
	}

	info := FileSystemInfo{
		Path:        req.Path,
		Name:        stat.Name(),
		Type:        itemType,
		Size:        stat.Size(),
		Permissions: stat.Mode().String(),
		Mode:        uint32(stat.Mode()),
		Modified:    stat.ModTime(),
		Accessed:    stat.ModTime(),
		IsSymlink:   stat.Mode()&os.ModeSymlink != 0,
	}

	if stat.IsDir() {
		count, totalSize := countDirContents(req.Path)
		info.FileCount = count
		info.TotalSize = totalSize
	}

	return info, nil
}

func countDirContents(dirPath string) (int, int64) {
	count := 0
	totalSize := int64(0)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return 0, 0
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		count++
		if !entry.IsDir() {
			totalSize += info.Size()
		}
	}

	return count, totalSize
}

func (t *InfoTool) Title() string {
	return "Get File Information"
}

func (t *InfoTool) Annotations() map[string]bool {
	return tools.ReadOnlyAnnotations()
}
