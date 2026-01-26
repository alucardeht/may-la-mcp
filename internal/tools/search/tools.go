package search

import (
	"github.com/alucardeht/may-la-mcp/internal/router"
	"github.com/alucardeht/may-la-mcp/internal/tools"
)

func GetTools(r *router.Router) []tools.Tool {
	return []tools.Tool{
		&SearchTool{},
		&FindTool{},
		NewSymbolsTool(r),
		NewReferencesTool(r),
	}
}

func GetToolByName(name string, r *router.Router) tools.Tool {
	for _, tool := range GetTools(r) {
		if tool.Name() == name {
			return tool
		}
	}
	return nil
}
