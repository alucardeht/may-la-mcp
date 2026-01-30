package files

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompleteWorkflow(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	createTool := &CreateTool{}
	createDirReq := CreateRequest{
		Path: filepath.Join(tempDir, "project"),
		Type: "dir",
	}
	createDirData, _ := json.Marshal(createDirReq)
	_, err := createTool.Execute(ctx, createDirData)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	testFile := filepath.Join(tempDir, "project", "main.go")
	writeReq := WriteRequest{
		Path:    testFile,
		Content: "package main\n\nfunc main() {\n  println(\"Hello\")\n}",
	}
	writeTool := &WriteTool{}
	writeData, _ := json.Marshal(writeReq)
	_, err = writeTool.Execute(ctx, writeData)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	infoReq := InfoRequest{Path: testFile}
	infoTool := &InfoTool{}
	infoData, _ := json.Marshal(infoReq)
	infoResult, err := infoTool.Execute(ctx, infoData)
	if err != nil {
		t.Fatalf("Failed to get info: %v", err)
	}
	infoResp := infoResult.(FileSystemInfo)
	if infoResp.Type != "file" {
		t.Errorf("Expected type file, got %s", infoResp.Type)
	}

	listReq := ListRequest{
		Path:    filepath.Join(tempDir, "project"),
		Pattern: "*.go",
	}
	listTool := &ListTool{}
	listData, _ := json.Marshal(listReq)
	listResult, err := listTool.Execute(ctx, listData)
	if err != nil {
		t.Fatalf("Failed to list directory: %v", err)
	}
	listResp := listResult.(ListResponse)
	if listResp.Count != 1 {
		t.Errorf("Expected 1 file, got %d", listResp.Count)
	}

	editReq := EditRequest{
		Path: testFile,
		Edits: []EditOperation{
			{
				Search:  "Hello",
				Replace: "May-la MCP",
			},
		},
	}
	editTool := &EditTool{}
	editData, _ := json.Marshal(editReq)
	_, err = editTool.Execute(ctx, editData)
	if err != nil {
		t.Fatalf("Failed to edit file: %v", err)
	}

	readReq := ReadRequest{Path: testFile}
	readTool := &ReadTool{}
	readData, _ := json.Marshal(readReq)
	readResult, err := readTool.Execute(ctx, readData)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	readResp := readResult.(ReadResponse)
	if !strings.Contains(readResp.Content, "May-la MCP") {
		t.Error("Edit operation did not work")
	}

	backupFile := filepath.Join(tempDir, "project", "main.go.bak")
	writeBackupReq := WriteRequest{
		Path:    testFile,
		Content: "new content",
		Backup:  true,
	}
	writeBackupData, _ := json.Marshal(writeBackupReq)
	writeBackupResp, err := writeTool.Execute(ctx, writeBackupData)
	if err != nil {
		t.Fatalf("Failed to write with backup: %v", err)
	}
	writeResp := writeBackupResp.(WriteResponse)
	if writeResp.Backup == "" {
		t.Error("Backup was not created")
	}

	if _, err := os.Stat(backupFile); err != nil {
		t.Error("Backup file not found")
	}

	moveReq := MoveRequest{
		Source:      testFile,
		Destination: filepath.Join(tempDir, "project", "app.go"),
	}
	moveTool := &MoveTool{}
	moveData, _ := json.Marshal(moveReq)
	_, err = moveTool.Execute(ctx, moveData)
	if err != nil {
		t.Fatalf("Failed to move file: %v", err)
	}

	if _, err := os.Stat(testFile); err == nil {
		t.Error("Original file still exists after move")
	}

	if _, err := os.Stat(filepath.Join(tempDir, "project", "app.go")); err != nil {
		t.Error("Moved file does not exist")
	}

	deleteReq := DeleteRequest{
		Path: filepath.Join(tempDir, "project"),
		Recursive: true,
	}
	deleteTool := &DeleteTool{}
	deleteData, _ := json.Marshal(deleteReq)
	_, err = deleteTool.Execute(ctx, deleteData)
	if err != nil {
		t.Fatalf("Failed to delete directory: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tempDir, "project")); err == nil {
		t.Error("Directory still exists after delete")
	}
}

func TestErrorHandling(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name      string
		tool      string
		input     interface{}
		shouldErr bool
	}{
		{
			name:      "read non-existent file",
			tool:      "read",
			input:     ReadRequest{Path: "/nonexistent/file.txt"},
			shouldErr: true,
		},
		{
			name:      "write empty path",
			tool:      "write",
			input:     WriteRequest{Path: "", Content: "test"},
			shouldErr: true,
		},
		{
			name:      "create invalid type",
			tool:      "create",
			input:     CreateRequest{Path: "/tmp/test", Type: "invalid"},
			shouldErr: true,
		},
		{
			name:      "delete non-existent",
			tool:      "delete",
			input:     DeleteRequest{Path: "/nonexistent"},
			shouldErr: true,
		},
		{
			name:      "list non-existent",
			tool:      "list",
			input:     ListRequest{Path: "/nonexistent"},
			shouldErr: true,
		},
		{
			name:      "info non-existent",
			tool:      "info",
			input:     InfoRequest{Path: "/nonexistent"},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, _ := json.Marshal(tt.input)

			var result interface{}
			var err error

			switch tt.tool {
			case "read":
				result, err = (&ReadTool{}).Execute(ctx, data)
			case "write":
				result, err = (&WriteTool{}).Execute(ctx, data)
			case "create":
				result, err = (&CreateTool{}).Execute(ctx, data)
			case "delete":
				result, err = (&DeleteTool{}).Execute(ctx, data)
			case "list":
				result, err = (&ListTool{}).Execute(ctx, data)
			case "info":
				result, err = (&InfoTool{}).Execute(ctx, data)
			}

			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected error, got nil (result: %v)", result)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestEditEdgeCases(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
	os.WriteFile(testFile, []byte(content), 0644)

	editTool := &EditTool{}

	t.Run("invalid line range", func(t *testing.T) {
		editReq := EditRequest{
			Path: testFile,
			Edits: []EditOperation{
				{StartLine: 10, EndLine: 20, NewContent: "invalid"},
			},
		}
		editData, _ := json.Marshal(editReq)
		_, err := editTool.Execute(ctx, editData)
		if err == nil {
			t.Error("Expected error for invalid line range")
		}
	})

	t.Run("empty edits", func(t *testing.T) {
		editReq := EditRequest{
			Path:  testFile,
			Edits: []EditOperation{},
		}
		editData, _ := json.Marshal(editReq)
		_, err := editTool.Execute(ctx, editData)
		if err == nil {
			t.Error("Expected error for empty edits")
		}
	})

	t.Run("replace multiple occurrences", func(t *testing.T) {
		editReq := EditRequest{
			Path: testFile,
			Edits: []EditOperation{
				{Search: "Line", Replace: "Row"},
			},
		}
		editData, _ := json.Marshal(editReq)
		_, err := editTool.Execute(ctx, editData)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

func TestEncoding(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		filename string
		content  string
		expected string
	}{
		{
			name:     "utf-8 content",
			filename: "utf8.txt",
			content:  "Hello, 世界",
			expected: "utf-8",
		},
		{
			name:     "ascii content",
			filename: "ascii.txt",
			content:  "Hello, World!",
			expected: "utf-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tempDir, tt.filename)
			os.WriteFile(testFile, []byte(tt.content), 0644)

			readTool := &ReadTool{}
			readReq := ReadRequest{
				Path:     testFile,
				Encoding: "auto",
			}
			readData, _ := json.Marshal(readReq)
			result, err := readTool.Execute(ctx, readData)
			if err != nil {
				t.Fatalf("Read failed: %v", err)
			}

			readResp := result.(ReadResponse)
			if readResp.Encoding != tt.expected {
				t.Errorf("Expected encoding %s, got %s", tt.expected, readResp.Encoding)
			}

			if readResp.Content != tt.content {
				t.Errorf("Content mismatch: got %q, want %q", readResp.Content, tt.content)
			}
		})
	}
}
