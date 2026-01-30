package files

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alucardeht/may-la-mcp/internal/tools"
)

type MoveRequest struct {
	Source    string `json:"source"`
	Destination string `json:"destination"`
	Overwrite bool   `json:"overwrite,omitempty"`
}

type MoveResponse struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Type        string `json:"type"`
	Size        int64  `json:"size"`
}

type MoveTool struct{}

func (t *MoveTool) Name() string {
	return "move"
}

func (t *MoveTool) Description() string {
	return "Move or rename file or directory"
}

func (t *MoveTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"source": {
				"type": "string",
				"description": "Source path (absolute path required)"
			},
			"destination": {
				"type": "string",
				"description": "Destination path (absolute path required)"
			},
			"overwrite": {
				"type": "boolean",
				"description": "Overwrite destination if exists (default: false)"
			}
		},
		"required": ["source", "destination"]
	}`)
}

func (t *MoveTool) Execute(ctx context.Context, input json.RawMessage) (interface{}, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var req MoveRequest
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if req.Source == "" {
		return nil, fmt.Errorf("source is required")
	}

	if req.Destination == "" {
		return nil, fmt.Errorf("destination is required")
	}

	sourceStat, err := os.Stat(req.Source)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("source does not exist")
		}
		return nil, fmt.Errorf("failed to stat source: %w", err)
	}

	destStat, err := os.Stat(req.Destination)
	if err == nil {
		if !req.Overwrite {
			return nil, fmt.Errorf("destination already exists, use overwrite=true")
		}

		if sourceStat.IsDir() != destStat.IsDir() {
			return nil, fmt.Errorf("source and destination types do not match")
		}

		if !sourceStat.IsDir() {
			if err := os.Remove(req.Destination); err != nil {
				return nil, fmt.Errorf("failed to remove existing destination: %w", err)
			}
		}
	}

	destDir := filepath.Dir(req.Destination)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	if err := os.Rename(req.Source, req.Destination); err != nil {
		return nil, fmt.Errorf("failed to move: %w", err)
	}

	newStat, err := os.Stat(req.Destination)
	itemType := "file"
	var size int64
	if err == nil {
		if newStat.IsDir() {
			itemType = "dir"
		}
		size = newStat.Size()
	}

	return MoveResponse{
		Source:      req.Source,
		Destination: req.Destination,
		Type:        itemType,
		Size:        size,
	}, nil
}

func (t *MoveTool) Title() string {
	return "Move or Rename File"
}

func (t *MoveTool) Annotations() map[string]bool {
	return tools.NonIdempotentWriteAnnotations()
}
