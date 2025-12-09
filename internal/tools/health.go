package tools

import "encoding/json"

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

func (t *HealthTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {},
		"required": []
	}`)
}

func (t *HealthTool) Execute(input json.RawMessage) (interface{}, error) {
	return map[string]interface{}{
		"status": "healthy",
		"tools":  "loaded",
	}, nil
}
