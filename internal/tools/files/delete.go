package files

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/alucardeht/may-la-mcp/internal/tools"
)

type DeleteRequest struct {
	Path      string `json:"path"`
	Recursive bool   `json:"recursive,omitempty"`
	Force     bool   `json:"force,omitempty"`
}

type DeleteResponse struct {
	Path    string `json:"path"`
	Deleted bool   `json:"deleted"`
	Type    string `json:"type"`
	Size    int64  `json:"size"`
}

type DeleteTool struct{}

func (t *DeleteTool) Name() string {
	return "delete"
}

func (t *DeleteTool) Description() string {
	return "Delete file or directory"
}

func (t *DeleteTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Path to delete (absolute path required)"
			},
			"recursive": {
				"type": "boolean",
				"description": "Delete contents recursively"
			},
			"force": {
				"type": "boolean",
				"description": "Force deletion without prompting (default: false)"
			}
		},
		"required": ["path"]
	}`)
}

func (t *DeleteTool) Execute(ctx context.Context, input json.RawMessage) (interface{}, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var req DeleteRequest
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

	itemType := "file"
	size := stat.Size()

	if stat.IsDir() {
		itemType = "dir"
		size = 0

		if !req.Recursive && !req.Force {
			entries, err := os.ReadDir(req.Path)
			if err == nil && len(entries) > 0 {
				return nil, fmt.Errorf("directory not empty, use recursive=true to delete")
			}
		}

		if err := os.RemoveAll(req.Path); err != nil {
			return nil, fmt.Errorf("failed to delete directory: %w", err)
		}
	} else {
		if err := os.Remove(req.Path); err != nil {
			return nil, fmt.Errorf("failed to delete file: %w", err)
		}
	}

	return DeleteResponse{
		Path:    req.Path,
		Deleted: true,
		Type:    itemType,
		Size:    size,
	}, nil
}

func (t *DeleteTool) Title() string {
	return "Delete File or Directory"
}

func (t *DeleteTool) Annotations() map[string]bool {
	return tools.DestructiveAnnotations()
}
