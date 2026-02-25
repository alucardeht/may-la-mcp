package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/alucardeht/may-la-mcp/internal/logger"
)

var log = logger.ForComponent("tools")

type Tool interface {
	Name() string
	Description() string
	Schema() json.RawMessage
	Execute(ctx context.Context, input json.RawMessage) (interface{}, error)
}

type AnnotatedTool interface {
	Tool
	Title() string
	Annotations() map[string]bool
}

type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := tool.Name()
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool already registered: %s", name)
	}

	r.tools[name] = tool
	return nil
}

func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}

func (r *Registry) Execute(ctx context.Context, name string, input json.RawMessage) (result interface{}, err error) {
	tool, ok := r.Get(name)
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic in tool %s: %v", name, p)
		}
	}()

	return tool.Execute(ctx, input)
}

func (r *Registry) ExecuteWithTimeout(name string, input json.RawMessage, timeout time.Duration) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	type result struct {
		value interface{}
		err   error
	}

	resultChan := make(chan result, 1)

	go func() {
		value, err := r.Execute(ctx, name, input)
		select {
		case resultChan <- result{value, err}:
		default:
			log.Warn("tool execution completed after timeout", "tool", name)
		}
	}()

	select {
	case res := <-resultChan:
		return res.value, res.err
	case <-ctx.Done():
		return nil, fmt.Errorf("tool execution timeout after %v", timeout)
	}
}

func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, tool)
	}
	return result
}

func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}
