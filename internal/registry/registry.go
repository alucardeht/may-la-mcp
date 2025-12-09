package registry

import (
	"encoding/json"
	"fmt"
	"log"
)

type Tool interface {
	Name() string
	Description() string
	Schema() json.RawMessage
	Execute(input json.RawMessage) (interface{}, error)
}

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(tool Tool) error {
	if tool.Name() == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	if _, exists := r.tools[tool.Name()]; exists {
		return fmt.Errorf("tool '%s' already registered", tool.Name())
	}

	r.tools[tool.Name()] = tool
	log.Printf("Registered tool: %s", tool.Name())
	return nil
}

func (r *Registry) Get(name string) (Tool, error) {
	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found", name)
	}
	return tool, nil
}

func (r *Registry) List() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

func (r *Registry) Execute(name string, input json.RawMessage) (interface{}, error) {
	tool, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	return tool.Execute(input)
}

func (r *Registry) GetToolDefinitions() []map[string]interface{} {
	definitions := make([]map[string]interface{}, 0, len(r.tools))

	for _, tool := range r.tools {
		def := map[string]interface{}{
			"name":        tool.Name(),
			"description": tool.Description(),
			"inputSchema": tool.Schema(),
		}
		definitions = append(definitions, def)
	}

	return definitions
}
