package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/maylamcp/mayla/internal/tools"
	"github.com/maylamcp/mayla/internal/tools/files"
	"github.com/maylamcp/mayla/internal/tools/memory"
	"github.com/maylamcp/mayla/internal/tools/search"
	"github.com/maylamcp/mayla/internal/tools/spec"
)

func TestAllToolsE2E(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mayla-e2e-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("Registry_AllToolsRegistered", func(t *testing.T) {
		registry := tools.NewRegistry()

		registry.Register(tools.NewHealthTool())

		for _, tool := range files.GetTools() {
			registry.Register(tool)
		}
		for _, tool := range search.GetTools() {
			registry.Register(tool)
		}
		for _, tool := range spec.GetTools() {
			registry.Register(tool)
		}

		dbPath := filepath.Join(tmpDir, "test-memory.db")
		memTools, err := memory.GetTools(dbPath)
		if err != nil {
			t.Fatalf("Failed to get memory tools: %v", err)
		}
		for _, tool := range memTools {
			registry.Register(tool)
		}

		names := registry.Names()
		expectedCount := 22
		if len(names) != expectedCount {
			t.Errorf("Expected %d tools, got %d: %v", expectedCount, len(names), names)
		}

		t.Logf("✅ Registered %d tools: %v", len(names), names)
	})

	t.Run("Files_CreateReadWriteEditDelete", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "test.txt")

		createTool := &files.CreateTool{}
		input, _ := json.Marshal(map[string]interface{}{
			"path": testFile,
			"type": "file",
		})
		result, err := createTool.Execute(input)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		t.Logf("✅ Create: %v", result)

		writeTool := &files.WriteTool{}
		input, _ = json.Marshal(map[string]interface{}{
			"path":    testFile,
			"content": "Hello May-la MCP!\nLine 2\nLine 3",
		})
		result, err = writeTool.Execute(input)
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
		t.Logf("✅ Write: %v", result)

		readTool := &files.ReadTool{}
		input, _ = json.Marshal(map[string]interface{}{
			"path": testFile,
		})
		result, err = readTool.Execute(input)
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}
		t.Logf("✅ Read: %v", result)

		editTool := &files.EditTool{}
		input, _ = json.Marshal(map[string]interface{}{
			"path": testFile,
			"edits": []map[string]interface{}{
				{
					"search":  "May-la",
					"replace": "MAYLA",
				},
			},
		})
		result, err = editTool.Execute(input)
		if err != nil {
			t.Fatalf("Edit failed: %v", err)
		}
		t.Logf("✅ Edit: %v", result)

		content, _ := os.ReadFile(testFile)
		if string(content) != "Hello MAYLA MCP!\nLine 2\nLine 3\n" {
			t.Errorf("Edit didn't work correctly: %s", content)
		}

		infoTool := &files.InfoTool{}
		input, _ = json.Marshal(map[string]interface{}{
			"path": testFile,
		})
		result, err = infoTool.Execute(input)
		if err != nil {
			t.Fatalf("Info failed: %v", err)
		}
		t.Logf("✅ Info: %v", result)

		movedFile := filepath.Join(tmpDir, "moved.txt")
		moveTool := &files.MoveTool{}
		input, _ = json.Marshal(map[string]interface{}{
			"source":      testFile,
			"destination": movedFile,
		})
		result, err = moveTool.Execute(input)
		if err != nil {
			t.Fatalf("Move failed: %v", err)
		}
		t.Logf("✅ Move: %v", result)

		listTool := &files.ListTool{}
		input, _ = json.Marshal(map[string]interface{}{
			"path": tmpDir,
		})
		result, err = listTool.Execute(input)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		t.Logf("✅ List: %v", result)

		deleteTool := &files.DeleteTool{}
		input, _ = json.Marshal(map[string]interface{}{
			"path":  movedFile,
			"force": true,
		})
		result, err = deleteTool.Execute(input)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		t.Logf("✅ Delete: %v", result)
	})

	t.Run("Search_GrepFindSymbolsReferences", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "search-test")
		os.MkdirAll(testDir, 0755)

		goFile := filepath.Join(testDir, "sample.go")
		os.WriteFile(goFile, []byte(`package main

func HelloWorld() {
	println("Hello")
}

func main() {
	HelloWorld()
}
`), 0644)

		searchTool := &search.SearchTool{}
		input, _ := json.Marshal(map[string]interface{}{
			"path":    testDir,
			"pattern": "HelloWorld",
		})
		result, err := searchTool.Execute(input)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		t.Logf("✅ Search: %v", result)

		findTool := &search.FindTool{}
		input, _ = json.Marshal(map[string]interface{}{
			"path":    testDir,
			"pattern": "*.go",
		})
		result, err = findTool.Execute(input)
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}
		t.Logf("✅ Find: %v", result)

		symbolsTool := &search.SymbolsTool{}
		input, _ = json.Marshal(map[string]interface{}{
			"path": goFile,
		})
		result, err = symbolsTool.Execute(input)
		if err != nil {
			t.Fatalf("Symbols failed: %v", err)
		}
		t.Logf("✅ Symbols: %v", result)

		refsTool := &search.ReferencesTool{}
		input, _ = json.Marshal(map[string]interface{}{
			"path":   testDir,
			"symbol": "HelloWorld",
		})
		result, err = refsTool.Execute(input)
		if err != nil {
			t.Fatalf("References failed: %v", err)
		}
		t.Logf("✅ References: %v", result)
	})

	t.Run("Spec_InitGenerateValidateStatus", func(t *testing.T) {
		specDir := filepath.Join(tmpDir, "spec-test")
		os.MkdirAll(specDir, 0755)

		initTool := &spec.InitTool{}
		input, _ := json.Marshal(map[string]interface{}{
			"path": specDir,
		})
		result, err := initTool.Execute(input)
		if err != nil {
			t.Fatalf("Spec Init failed: %v", err)
		}
		t.Logf("✅ Spec Init: %v", result)

		generateTool := &spec.GenerateTool{}
		input, _ = json.Marshal(map[string]interface{}{
			"path":     specDir,
			"artifact": "constitution",
			"content": map[string]interface{}{
				"principles": []string{"Performance first", "Clean code"},
			},
		})
		result, err = generateTool.Execute(input)
		if err != nil {
			t.Fatalf("Spec Generate failed: %v", err)
		}
		t.Logf("✅ Spec Generate: %v", result)

		validateTool := &spec.ValidateTool{}
		input, _ = json.Marshal(map[string]interface{}{
			"path": specDir,
		})
		result, err = validateTool.Execute(input)
		if err != nil {
			t.Fatalf("Spec Validate failed: %v", err)
		}
		t.Logf("✅ Spec Validate: %v", result)

		statusTool := &spec.StatusTool{}
		input, _ = json.Marshal(map[string]interface{}{
			"path": specDir,
		})
		result, err = statusTool.Execute(input)
		if err != nil {
			t.Fatalf("Spec Status failed: %v", err)
		}
		t.Logf("✅ Spec Status: %v", result)
	})

	t.Run("Memory_WriteReadListSearchDelete", func(t *testing.T) {
		dbPath := filepath.Join(tmpDir, "memory-test.db")
		memTools, err := memory.GetTools(dbPath)
		if err != nil {
			t.Fatalf("Failed to initialize memory tools: %v", err)
		}

		var writeTool, readTool, listTool, searchTool, deleteTool tools.Tool
		for _, tool := range memTools {
			switch tool.Name() {
			case "memory_write":
				writeTool = tool
			case "memory_read":
				readTool = tool
			case "memory_list":
				listTool = tool
			case "memory_search":
				searchTool = tool
			case "memory_delete":
				deleteTool = tool
			}
		}

		input, _ := json.Marshal(map[string]interface{}{
			"name":     "test-memory",
			"content":  "This is a test memory for E2E testing",
			"category": "notes",
			"tags":     []string{"test", "e2e"},
		})
		result, err := writeTool.Execute(input)
		if err != nil {
			t.Fatalf("Memory Write failed: %v", err)
		}
		t.Logf("✅ Memory Write: %v", result)

		input, _ = json.Marshal(map[string]interface{}{
			"name": "test-memory",
		})
		result, err = readTool.Execute(input)
		if err != nil {
			t.Fatalf("Memory Read failed: %v", err)
		}
		t.Logf("✅ Memory Read: %v", result)

		input, _ = json.Marshal(map[string]interface{}{})
		result, err = listTool.Execute(input)
		if err != nil {
			t.Fatalf("Memory List failed: %v", err)
		}
		t.Logf("✅ Memory List: %v", result)

		input, _ = json.Marshal(map[string]interface{}{
			"query": "E2E testing",
		})
		result, err = searchTool.Execute(input)
		if err != nil {
			t.Fatalf("Memory Search failed: %v", err)
		}
		t.Logf("✅ Memory Search: %v", result)

		input, _ = json.Marshal(map[string]interface{}{
			"name": "test-memory",
		})
		result, err = deleteTool.Execute(input)
		if err != nil {
			t.Fatalf("Memory Delete failed: %v", err)
		}
		t.Logf("✅ Memory Delete: %v", result)
	})

	t.Run("Health_Check", func(t *testing.T) {
		healthTool := tools.NewHealthTool()
		result, err := healthTool.Execute(json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("Health failed: %v", err)
		}
		t.Logf("✅ Health: %v", result)
	})
}

func TestToolsIndividually(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mayla-individual-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fileTools := files.GetTools()
	for _, tool := range fileTools {
		t.Run("Tool_"+tool.Name(), func(t *testing.T) {
			if tool.Name() == "" {
				t.Error("Tool name is empty")
			}
			if tool.Description() == "" {
				t.Error("Tool description is empty")
			}
			if len(tool.Schema()) == 0 {
				t.Error("Tool schema is empty")
			}
			t.Logf("✅ %s: name=%s, desc=%d chars, schema=%d bytes",
				tool.Name(), tool.Name(), len(tool.Description()), len(tool.Schema()))
		})
	}

	searchTools := search.GetTools()
	for _, tool := range searchTools {
		t.Run("Tool_"+tool.Name(), func(t *testing.T) {
			if tool.Name() == "" {
				t.Error("Tool name is empty")
			}
			if tool.Description() == "" {
				t.Error("Tool description is empty")
			}
			if len(tool.Schema()) == 0 {
				t.Error("Tool schema is empty")
			}
			t.Logf("✅ %s validated", tool.Name())
		})
	}

	specTools := spec.GetTools()
	for _, tool := range specTools {
		t.Run("Tool_"+tool.Name(), func(t *testing.T) {
			if tool.Name() == "" {
				t.Error("Tool name is empty")
			}
			if tool.Description() == "" {
				t.Error("Tool description is empty")
			}
			if len(tool.Schema()) == 0 {
				t.Error("Tool schema is empty")
			}
			t.Logf("✅ %s validated", tool.Name())
		})
	}

	dbPath := filepath.Join(tmpDir, "validate-memory.db")
	memTools, _ := memory.GetTools(dbPath)
	for _, tool := range memTools {
		t.Run("Tool_"+tool.Name(), func(t *testing.T) {
			if tool.Name() == "" {
				t.Error("Tool name is empty")
			}
			if tool.Description() == "" {
				t.Error("Tool description is empty")
			}
			if len(tool.Schema()) == 0 {
				t.Error("Tool schema is empty")
			}
			t.Logf("✅ %s validated", tool.Name())
		})
	}
}

func TestErrorScenarios(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mayla-errors-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("Files_ReadNonexistent", func(t *testing.T) {
		readTool := &files.ReadTool{}
		input, _ := json.Marshal(map[string]interface{}{
			"path": filepath.Join(tmpDir, "nonexistent.txt"),
		})
		_, err := readTool.Execute(input)
		if err == nil {
			t.Error("Expected error when reading nonexistent file")
		}
		t.Logf("✅ ReadNonexistent: correctly returned error")
	})


	t.Run("Files_DeleteWithoutForce", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "protected.txt")
		os.WriteFile(testFile, []byte("protected"), 0644)

		deleteTool := &files.DeleteTool{}
		input, _ := json.Marshal(map[string]interface{}{
			"path":  testFile,
			"force": false,
		})
		result, err := deleteTool.Execute(input)
		t.Logf("✅ DeleteWithoutForce: returned result=%v, err=%v", result, err)
	})

	t.Run("Memory_ReadNonexistent", func(t *testing.T) {
		dbPath := filepath.Join(tmpDir, "memory-error.db")
		memTools, _ := memory.GetTools(dbPath)

		var readTool tools.Tool
		for _, tool := range memTools {
			if tool.Name() == "memory_read" {
				readTool = tool
				break
			}
		}

		if readTool != nil {
			input, _ := json.Marshal(map[string]interface{}{
				"name": "nonexistent-memory",
			})
			_, err := readTool.Execute(input)
			if err == nil {
				t.Error("Expected error when reading nonexistent memory")
			}
			t.Logf("✅ MemoryReadNonexistent: correctly returned error")
		}
	})

	t.Run("Spec_InitInvalidPath", func(t *testing.T) {
		initTool := &spec.InitTool{}
		input, _ := json.Marshal(map[string]interface{}{
			"path": "/invalid/path/that/does/not/exist",
		})
		_, err := initTool.Execute(input)
		if err == nil {
			t.Error("Expected error with invalid path")
		}
		t.Logf("✅ SpecInitInvalidPath: correctly returned error")
	})
}

func TestToolMetadata(t *testing.T) {
	t.Run("AllTools_HaveValidMetadata", func(t *testing.T) {
		fileTools := files.GetTools()
		searchTools := search.GetTools()
		specTools := spec.GetTools()
		healthTool := tools.NewHealthTool()

		allTools := make([]tools.Tool, 0)
		allTools = append(allTools, fileTools...)
		allTools = append(allTools, searchTools...)
		allTools = append(allTools, specTools...)
		allTools = append(allTools, healthTool)

		for _, tool := range allTools {
			if tool.Name() == "" {
				t.Errorf("Tool has empty name")
			}
			if tool.Description() == "" {
				t.Errorf("Tool %s has empty description", tool.Name())
			}
			schema := tool.Schema()
			if len(schema) == 0 {
				t.Errorf("Tool %s has empty schema", tool.Name())
			}

			var schemaMap map[string]interface{}
			err := json.Unmarshal(schema, &schemaMap)
			if err != nil {
				t.Errorf("Tool %s has invalid JSON schema: %v", tool.Name(), err)
			}
		}

		t.Logf("✅ All %d tools have valid metadata", len(allTools))
	})
}

func TestMemoryIntegration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mayla-memory-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "memory-integration.db")
	memTools, err := memory.GetTools(dbPath)
	if err != nil {
		t.Fatalf("Failed to get memory tools: %v", err)
	}

	var writeMemory, readMemory, listMemory, searchMemory, deleteMemory tools.Tool

	for _, tool := range memTools {
		switch tool.Name() {
		case "memory_write":
			writeMemory = tool
		case "memory_read":
			readMemory = tool
		case "memory_list":
			listMemory = tool
		case "memory_search":
			searchMemory = tool
		case "memory_delete":
			deleteMemory = tool
		}
	}

	if writeMemory == nil || readMemory == nil || listMemory == nil || searchMemory == nil || deleteMemory == nil {
		t.Fatal("Not all memory tools found")
	}

	t.Run("Memory_FullLifecycle", func(t *testing.T) {
		writeInput, _ := json.Marshal(map[string]interface{}{
			"name":     "lifecycle-test",
			"content":  "Testing full memory lifecycle",
			"category": "testing",
			"tags":     []string{"integration", "lifecycle"},
		})
		result, err := writeMemory.Execute(writeInput)
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
		t.Logf("✅ Write: %v", result)

		readInput, _ := json.Marshal(map[string]interface{}{
			"name": "lifecycle-test",
		})
		result, err = readMemory.Execute(readInput)
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}
		t.Logf("✅ Read: %v", result)

		listInput, _ := json.Marshal(map[string]interface{}{})
		result, err = listMemory.Execute(listInput)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		t.Logf("✅ List: %v", result)

		searchInput, _ := json.Marshal(map[string]interface{}{
			"query": "lifecycle",
		})
		result, err = searchMemory.Execute(searchInput)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		t.Logf("✅ Search: %v", result)

		deleteInput, _ := json.Marshal(map[string]interface{}{
			"name": "lifecycle-test",
		})
		result, err = deleteMemory.Execute(deleteInput)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		t.Logf("✅ Delete: %v", result)
	})

	t.Run("Memory_MultipleRecords", func(t *testing.T) {
		records := []map[string]interface{}{
			{
				"name":     "record-1",
				"content":  "First record",
				"category": "multi-test",
				"tags":     []string{"first"},
			},
			{
				"name":     "record-2",
				"content":  "Second record",
				"category": "multi-test",
				"tags":     []string{"second"},
			},
			{
				"name":     "record-3",
				"content":  "Third record",
				"category": "multi-test",
				"tags":     []string{"third"},
			},
		}

		for _, record := range records {
			input, _ := json.Marshal(record)
			_, err := writeMemory.Execute(input)
			if err != nil {
				t.Fatalf("Write failed for %s: %v", record["name"], err)
			}
		}

		listInput, _ := json.Marshal(map[string]interface{}{})
		result, err := listMemory.Execute(listInput)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		t.Logf("✅ MultipleRecords: %v", result)

		for _, record := range records {
			deleteInput, _ := json.Marshal(map[string]interface{}{
				"name": record["name"],
			})
			_, err := deleteMemory.Execute(deleteInput)
			if err != nil {
				t.Fatalf("Delete failed for %s: %v", record["name"], err)
			}
		}
	})
}
