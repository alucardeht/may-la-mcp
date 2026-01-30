package examples

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/alucardeht/may-la-mcp/internal/registry"
	"log"
)

func ExampleFileOperations() {
	ctx := context.Background()
	reg := registry.NewRegistry()

	if err := registry.InitializeAllTools(reg); err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Write Tool ===")
	writeInput := map[string]interface{}{
		"path":    "/tmp/example.txt",
		"content": "Hello, May-la MCP!\nLine 2\nLine 3",
		"backup":  false,
	}
	writeData, _ := json.Marshal(writeInput)
	result, err := reg.Execute(ctx, "write", writeData)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Write result: %+v\n", result)
	}

	fmt.Println("\n=== Read Tool ===")
	readInput := map[string]interface{}{
		"path": "/tmp/example.txt",
	}
	readData, _ := json.Marshal(readInput)
	result, err = reg.Execute(ctx, "read", readData)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Read result: %+v\n", result)
	}

	fmt.Println("\n=== Edit Tool ===")
	editInput := map[string]interface{}{
		"path": "/tmp/example.txt",
		"edits": []map[string]interface{}{
			{
				"startLine":  2,
				"endLine":    2,
				"newContent": "Modified Line 2",
			},
		},
	}
	editData, _ := json.Marshal(editInput)
	result, err = reg.Execute(ctx, "edit", editData)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Edit result: %+v\n", result)
	}

	fmt.Println("\n=== List Tool ===")
	listInput := map[string]interface{}{
		"path":      "/tmp",
		"pattern":   "*.txt",
		"sortBy":    "name",
	}
	listData, _ := json.Marshal(listInput)
	result, err = reg.Execute(ctx, "list", listData)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		listResp := result.(map[string]interface{})
		fmt.Printf("Listed %d files\n", listResp["count"])
	}

	fmt.Println("\n=== Info Tool ===")
	infoInput := map[string]interface{}{
		"path": "/tmp/example.txt",
	}
	infoData, _ := json.Marshal(infoInput)
	result, err = reg.Execute(ctx, "info", infoData)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Info result: %+v\n", result)
	}

	fmt.Println("\n=== Available Tools ===")
	tools := reg.List()
	for _, tool := range tools {
		fmt.Printf("- %s: %s\n", tool.Name(), tool.Description())
	}
}
