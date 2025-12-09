package search

import (
	"github.com/alucardeht/may-la-mcp/internal/tools"
)

func GetTools() []tools.Tool {
	return []tools.Tool{
		&SearchTool{},
		&FindTool{},
		&SymbolsTool{},
		&ReferencesTool{},
	}
}

func GetToolByName(name string) tools.Tool {
	for _, tool := range GetTools() {
		if tool.Name() == name {
			return tool
		}
	}
	return nil
}
