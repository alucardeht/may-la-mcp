package registry

import (
	"github.com/maylamcp/mayla/internal/tools/files"
	"log"
)

func InitializeFileTools(registry *Registry) error {
	tools := files.GetTools()

	for _, tool := range tools {
		if err := registry.Register(tool); err != nil {
			log.Printf("Failed to register file tool: %v", err)
			return err
		}
	}

	log.Printf("Successfully registered %d file tools", len(tools))
	return nil
}

func InitializeAllTools(registry *Registry) error {
	if err := InitializeFileTools(registry); err != nil {
		return err
	}

	return nil
}
