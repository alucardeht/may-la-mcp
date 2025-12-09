package files

import (
	"github.com/maylamcp/mayla/internal/tools"
)

func GetTools() []tools.Tool {
	return []tools.Tool{
		&ReadTool{},
		&WriteTool{},
		&EditTool{},
		&CreateTool{},
		&DeleteTool{},
		&MoveTool{},
		&ListTool{},
		&InfoTool{},
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
