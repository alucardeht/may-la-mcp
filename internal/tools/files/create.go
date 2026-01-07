package files

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type CreateRequest struct {
	Path    string `json:"path"`
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
	Mode    string `json:"mode,omitempty"`
	Force   bool   `json:"force,omitempty"`
}

type CreateResponse struct {
	Path    string `json:"path"`
	Type    string `json:"type"`
	Created bool   `json:"created"`
	Size    int64  `json:"size"`
}

type CreateTool struct{}

func (t *CreateTool) Name() string {
	return "create"
}

func (t *CreateTool) Description() string {
	return "Create new file or directory"
}

func (t *CreateTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Path to create (absolute path required)"
			},
			"type": {
				"type": "string",
				"description": "Type of item to create",
				"enum": ["file", "dir"]
			},
			"content": {
				"type": "string",
				"description": "Initial content for file (optional)"
			},
			"mode": {
				"type": "string",
				"description": "Octal permissions (default: 0644)"
			},
			"force": {
				"type": "boolean",
				"description": "Overwrite if exists (default: false)"
			}
		},
		"required": ["path", "type"]
	}`)
}

func (t *CreateTool) Execute(input json.RawMessage) (interface{}, error) {
	var req CreateRequest
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if req.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	if req.Type != "file" && req.Type != "dir" {
		return nil, fmt.Errorf("type must be 'file' or 'dir'")
	}

	stat, err := os.Stat(req.Path)
	if err == nil {
		if !req.Force {
			return nil, fmt.Errorf("path already exists")
		}
		if req.Type == "dir" && !stat.IsDir() {
			return nil, fmt.Errorf("path exists and is not a directory")
		}
		if req.Type == "file" && stat.IsDir() {
			return nil, fmt.Errorf("path exists and is not a file")
		}
	}

	if req.Type == "dir" {
		var mode os.FileMode = 0755
		if req.Mode != "" {
			parsedMode, err := parseMode(req.Mode)
			if err != nil {
				return nil, fmt.Errorf("invalid mode: %w", err)
			}
			mode = parsedMode
		}

		if err := os.MkdirAll(req.Path, mode); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}

		return CreateResponse{
			Path:    req.Path,
			Type:    "dir",
			Created: true,
			Size:    0,
		}, nil
	}

	dir := filepath.Dir(req.Path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create parent directories: %w", err)
		}
	}

	var mode os.FileMode = 0644
	if req.Mode != "" {
		parsedMode, err := parseMode(req.Mode)
		if err != nil {
			return nil, fmt.Errorf("invalid mode: %w", err)
		}
		mode = parsedMode
	}

	if err := os.WriteFile(req.Path, []byte(req.Content), mode); err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	fileStat, _ := os.Stat(req.Path)
	return CreateResponse{
		Path:    req.Path,
		Type:    "file",
		Created: true,
		Size:    fileStat.Size(),
	}, nil
}

func parseMode(modeStr string) (os.FileMode, error) {
	var mode os.FileMode
	_, err := fmt.Sscanf(modeStr, "%o", &mode)
	if err != nil {
		return 0, err
	}
	return mode, nil
}
