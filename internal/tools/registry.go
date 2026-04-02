package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

type Tool interface {
	Name() string
	Description() string
	Schema() json.RawMessage
	Annotations() map[string]bool
	Execute(ctx context.Context, input json.RawMessage) (interface{}, error)
}

type Registry struct {
	tools map[string]Tool
	order []string
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
	r.order = append(r.order, tool.Name())
}

func (r *Registry) Execute(ctx context.Context, name string, input json.RawMessage) (interface{}, error) {
	tool, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return tool.Execute(ctx, input)
}

func (r *Registry) List() []Tool {
	result := make([]Tool, 0, len(r.order))
	for _, name := range r.order {
		if tool, ok := r.tools[name]; ok {
			result = append(result, tool)
		}
	}
	return result
}
