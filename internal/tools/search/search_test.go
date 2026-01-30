package search

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSearchTool(t *testing.T) {
	tool := &SearchTool{}

	if tool.Name() != "search" {
		t.Errorf("expected name 'search', got '%s'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("description should not be empty")
	}

	schema := tool.Schema()
	if len(schema) == 0 {
		t.Error("schema should not be empty")
	}
}

func TestFindTool(t *testing.T) {
	tool := &FindTool{}

	if tool.Name() != "find" {
		t.Errorf("expected name 'find', got '%s'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("description should not be empty")
	}

	schema := tool.Schema()
	if len(schema) == 0 {
		t.Error("schema should not be empty")
	}
}

func TestSymbolsTool(t *testing.T) {
	tool := &SymbolsTool{}

	if tool.Name() != "symbols" {
		t.Errorf("expected name 'symbols', got '%s'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("description should not be empty")
	}

	schema := tool.Schema()
	if len(schema) == 0 {
		t.Error("schema should not be empty")
	}
}

func TestReferencesTool(t *testing.T) {
	tool := &ReferencesTool{}

	if tool.Name() != "references" {
		t.Errorf("expected name 'references', got '%s'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("description should not be empty")
	}

	schema := tool.Schema()
	if len(schema) == 0 {
		t.Error("schema should not be empty")
	}
}

func TestGetTools(t *testing.T) {
	tools := GetTools(nil)

	if len(tools) != 4 {
		t.Errorf("expected 4 tools, got %d", len(tools))
	}

	names := []string{"search", "find", "symbols", "references"}
	for i, expectedName := range names {
		if tools[i].Name() != expectedName {
			t.Errorf("expected tool %d to be '%s', got '%s'", i, expectedName, tools[i].Name())
		}
	}
}

func TestGetToolByName(t *testing.T) {
	searchTool := GetToolByName("search", nil)
	if searchTool == nil {
		t.Error("search tool should not be nil")
	}

	findTool := GetToolByName("find", nil)
	if findTool == nil {
		t.Error("find tool should not be nil")
	}

	symbolsTool := GetToolByName("symbols", nil)
	if symbolsTool == nil {
		t.Error("symbols tool should not be nil")
	}

	referencesTool := GetToolByName("references", nil)
	if referencesTool == nil {
		t.Error("references tool should not be nil")
	}

	nonExistent := GetToolByName("nonexistent", nil)
	if nonExistent != nil {
		t.Error("nonexistent tool should be nil")
	}
}

func TestSearchWithGo(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	testFile := filepath.Join(tempDir, "test.txt")
	content := "hello world\nfoo bar\nhello again"
	os.WriteFile(testFile, []byte(content), 0644)

	req := SearchRequest{
		Pattern: "hello",
		Path:    tempDir,
	}

	resp, err := searchWithGo(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	searchResp := resp.(*SearchResponse)
	if searchResp.Count != 2 {
		t.Errorf("expected 2 matches, got %d", searchResp.Count)
	}

	if len(searchResp.Matches) != 2 {
		t.Errorf("expected 2 matches in array, got %d", len(searchResp.Matches))
	}

	if searchResp.Matches[0].Line != 1 {
		t.Errorf("expected first match on line 1, got %d", searchResp.Matches[0].Line)
	}

	if searchResp.Matches[1].Line != 3 {
		t.Errorf("expected second match on line 3, got %d", searchResp.Matches[1].Line)
	}
}

func TestSearchRequestValidation(t *testing.T) {
	tool := &SearchTool{}
	ctx := context.Background()

	invalidJSON := json.RawMessage(`{"invalid": "json"`)
	_, err := tool.Execute(ctx, invalidJSON)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}

	noPattern := json.RawMessage(`{"path": "/tmp"}`)
	_, err = tool.Execute(ctx, noPattern)
	if err == nil {
		t.Error("expected error for missing pattern")
	}

	noPath := json.RawMessage(`{"pattern": "test"}`)
	_, err = tool.Execute(ctx, noPath)
	if err == nil {
		t.Error("expected error for missing path")
	}
}

func TestFindWithGlob(t *testing.T) {
	tempDir := t.TempDir()

	os.WriteFile(filepath.Join(tempDir, "file1.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tempDir, "file2.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tempDir, "file3.txt"), []byte(""), 0644)

	req := FindRequest{
		Pattern: "*.go",
		Path:    tempDir,
	}

	resp, err := searchWithGoFind(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	findResp := resp.(*FindResponse)
	if findResp.Count != 2 {
		t.Errorf("expected 2 files, got %d", findResp.Count)
	}
}

func searchWithGoFind(req FindRequest) (interface{}, error) {
	if req.MaxResults == 0 {
		req.MaxResults = 1000
	}
	if req.Type == "" {
		req.Type = "all"
	}

	files := []FileInfo{}
	totalSize := int64(0)

	err := filepath.WalkDir(req.Path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if len(files) >= req.MaxResults {
			return filepath.SkipDir
		}

		relPath, err := filepath.Rel(req.Path, path)
		if err != nil {
			return nil
		}

		if matchesPattern(relPath, req.Pattern) {
			if shouldInclude(d, req.Type) {
				info, err := d.Info()
				if err != nil {
					return nil
				}

				fileType := "file"
				if d.IsDir() {
					fileType = "dir"
				}

				files = append(files, FileInfo{
					Path:     path,
					Type:     fileType,
					Size:     info.Size(),
					Modified: info.ModTime(),
				})
				totalSize += info.Size()

				if len(files) >= req.MaxResults {
					return filepath.SkipDir
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &FindResponse{
		Files:  files,
		Count:  len(files),
		Path:   req.Path,
		Total:  totalSize,
	}, nil
}
