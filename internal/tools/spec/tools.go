package spec

import (
	"encoding/json"

	"github.com/maylamcp/mayla/internal/tools"
)

func GetTools() []tools.Tool {
	return []tools.Tool{
		&InitTool{},
		&GenerateTool{},
		&ValidateTool{},
		&StatusTool{},
	}
}

type InitTool struct{}

func (t *InitTool) Name() string {
	return "spec_init"
}

func (t *InitTool) Description() string {
	return "Initialize a May-la spec-driven project structure"
}

func (t *InitTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Project path to initialize"
			},
			"force": {
				"type": "boolean",
				"description": "Overwrite existing files"
			}
		},
		"required": ["path"]
	}`)
}

func (t *InitTool) Execute(input json.RawMessage) (interface{}, error) {
	var req struct {
		Path  string `json:"path"`
		Force bool   `json:"force"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, err
	}
	return Init(req.Path, req.Force)
}

type GenerateTool struct{}

func (t *GenerateTool) Name() string {
	return "spec_generate"
}

func (t *GenerateTool) Description() string {
	return "Generate spec artifacts (constitution, spec, plan, tasks)"
}

func (t *GenerateTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Project path"
			},
			"artifact": {
				"type": "string",
				"enum": ["constitution", "spec", "plan", "tasks"],
				"description": "Artifact type to generate"
			},
			"content": {
				"type": "object",
				"description": "Content for the artifact"
			}
		},
		"required": ["path", "artifact"]
	}`)
}

func (t *GenerateTool) Execute(input json.RawMessage) (interface{}, error) {
	var req struct {
		Path     string                 `json:"path"`
		Artifact string                 `json:"artifact"`
		Content  map[string]interface{} `json:"content"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, err
	}
	return Generate(req.Path, req.Artifact, req.Content)
}

type ValidateTool struct{}

func (t *ValidateTool) Name() string {
	return "spec_validate"
}

func (t *ValidateTool) Description() string {
	return "Validate spec artifacts for consistency"
}

func (t *ValidateTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Project path"
			}
		},
		"required": ["path"]
	}`)
}

func (t *ValidateTool) Execute(input json.RawMessage) (interface{}, error) {
	var req struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, err
	}
	return Validate(req.Path)
}

type StatusTool struct{}

func (t *StatusTool) Name() string {
	return "spec_status"
}

func (t *StatusTool) Description() string {
	return "Get status of spec-driven project"
}

func (t *StatusTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Project path"
			}
		},
		"required": ["path"]
	}`)
}

func (t *StatusTool) Execute(input json.RawMessage) (interface{}, error) {
	var req struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, err
	}
	return Status(req.Path)
}
