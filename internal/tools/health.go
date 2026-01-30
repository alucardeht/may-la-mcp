package tools

import (
	"context"
	"encoding/json"
	"os"
)

type HealthTool struct{}

func NewHealthTool() *HealthTool {
	return &HealthTool{}
}

func (t *HealthTool) Name() string {
	return "health"
}

func (t *HealthTool) Description() string {
	return "Check daemon health status"
}

func (t *HealthTool) Title() string {
	return "Check Daemon Health"
}

func (t *HealthTool) Annotations() map[string]bool {
	return ReadOnlyAnnotations()
}

func (t *HealthTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {},
		"required": []
	}`)
}

func (t *HealthTool) Execute(ctx context.Context, input json.RawMessage) (interface{}, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	cwd, _ := os.Getwd()
	return map[string]interface{}{
		"status":    "healthy",
		"tools":     "loaded",
		"workspace": cwd,
	}, nil
}
