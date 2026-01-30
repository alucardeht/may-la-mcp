package files

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/alucardeht/may-la-mcp/internal/tools"
)

type WriteRequest struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	CreateDirs bool  `json:"createDirs,omitempty"`
	Backup    bool   `json:"backup,omitempty"`
}

type WriteResponse struct {
	Size    int64  `json:"size"`
	Path    string `json:"path"`
	Backup  string `json:"backup,omitempty"`
	Created bool   `json:"created"`
}

type WriteTool struct{}

func (t *WriteTool) Name() string {
	return "write"
}

func (t *WriteTool) Description() string {
	return "Write file contents with atomic operations and optional backup"
}

func (t *WriteTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Path to the file to write (absolute path required)"
			},
			"content": {
				"type": "string",
				"description": "File content to write"
			},
			"createDirs": {
				"type": "boolean",
				"description": "Create parent dirs if needed"
			},
			"backup": {
				"type": "boolean",
				"description": "Create backup .bak file before overwriting (default: false)"
			}
		},
		"required": ["path", "content"]
	}`)
}

func (t *WriteTool) Execute(input json.RawMessage) (interface{}, error) {
	var req WriteRequest
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if req.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	dir := filepath.Dir(req.Path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directories: %w", err)
		}
	}

	var backupPath string
	fileExists := false
	if stat, err := os.Stat(req.Path); err == nil && !stat.IsDir() {
		fileExists = true

		if req.Backup {
			backupPath = req.Path + ".bak." + strconv.FormatInt(time.Now().UnixNano(), 10)
			if err := os.Rename(req.Path, backupPath); err != nil {
				return nil, fmt.Errorf("failed to create backup: %w", err)
			}
		}
	}

	tempPath := req.Path + ".tmp." + strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := os.WriteFile(tempPath, []byte(req.Content), 0644); err != nil {
		if backupPath != "" {
			os.Rename(backupPath, req.Path)
		}
		return nil, fmt.Errorf("failed to write temporary file: %w", err)
	}

	if err := os.Rename(tempPath, req.Path); err != nil {
		os.Remove(tempPath)
		if backupPath != "" {
			os.Rename(backupPath, req.Path)
		}
		return nil, fmt.Errorf("failed to rename file: %w", err)
	}

	var size int64
	if stat, err := os.Stat(req.Path); err == nil {
		size = stat.Size()
	}

	return WriteResponse{
		Size:    size,
		Path:    req.Path,
		Backup:  backupPath,
		Created: !fileExists,
	}, nil
}

func (t *WriteTool) Title() string {
	return "Write File"
}

func (t *WriteTool) Annotations() map[string]bool {
	return tools.SafeWriteAnnotations()
}
