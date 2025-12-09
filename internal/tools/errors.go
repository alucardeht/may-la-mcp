package tools

import "fmt"

type ToolError struct {
	Code    int
	Message string
}

func (e *ToolError) Error() string {
	return e.Message
}

func NewToolNotFoundError(name string) *ToolError {
	return &ToolError{
		Code:    -32601,
		Message: fmt.Sprintf("Tool not found: %s", name),
	}
}

func NewToolExecutionError(name string, err error) *ToolError {
	return &ToolError{
		Code:    -32603,
		Message: fmt.Sprintf("Error executing tool %s: %v", name, err),
	}
}
