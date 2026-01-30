package files

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReadWrite(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := "Hello, World!\nLine 2\nLine 3"

	writeTool := &WriteTool{}
	writeReq := WriteRequest{
		Path:    testFile,
		Content: content,
	}

	writeData, _ := json.Marshal(writeReq)
	_, err := writeTool.Execute(ctx, writeData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	readTool := &ReadTool{}
	readReq := ReadRequest{
		Path: testFile,
	}

	readData, _ := json.Marshal(readReq)
	result, err := readTool.Execute(ctx, readData)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	readResp := result.(ReadResponse)
	if readResp.Content != content {
		t.Errorf("Content mismatch: got %q, want %q", readResp.Content, content)
	}

	if readResp.Lines != 3 {
		t.Errorf("Line count mismatch: got %d, want 3", readResp.Lines)
	}
}

func TestEdit(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	originalContent := "Line 1\nLine 2\nLine 3"

	os.WriteFile(testFile, []byte(originalContent), 0644)

	editTool := &EditTool{}
	editReq := EditRequest{
		Path: testFile,
		Edits: []EditOperation{
			{
				StartLine:  2,
				EndLine:    2,
				NewContent: "Modified Line 2",
			},
		},
	}

	editData, _ := json.Marshal(editReq)
	result, err := editTool.Execute(ctx, editData)
	if err != nil {
		t.Fatalf("Edit failed: %v", err)
	}

	editResp := result.(EditResponse)
	if !editResp.Modified {
		t.Error("Expected Modified to be true")
	}

	readTool := &ReadTool{}
	readReq := ReadRequest{Path: testFile}
	readData, _ := json.Marshal(readReq)
	readResult, _ := readTool.Execute(ctx, readData)
	readResp := readResult.(ReadResponse)

	if readResp.Content != "Line 1\nModified Line 2\nLine 3" {
		t.Errorf("Edit failed: got %q", readResp.Content)
	}
}

func TestCreateDelete(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "newfile.txt")

	createTool := &CreateTool{}
	createReq := CreateRequest{
		Path:    testFile,
		Type:    "file",
		Content: "test content",
	}

	createData, _ := json.Marshal(createReq)
	result, err := createTool.Execute(ctx, createData)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	createResp := result.(CreateResponse)
	if !createResp.Created {
		t.Error("Expected Created to be true")
	}

	if _, err := os.Stat(testFile); err != nil {
		t.Errorf("File not created: %v", err)
	}

	deleteTool := &DeleteTool{}
	deleteReq := DeleteRequest{
		Path: testFile,
	}

	deleteData, _ := json.Marshal(deleteReq)
	_, err = deleteTool.Execute(ctx, deleteData)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, err := os.Stat(testFile); err == nil {
		t.Error("File not deleted")
	}
}

func TestMove(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	srcFile := filepath.Join(tempDir, "source.txt")
	dstFile := filepath.Join(tempDir, "destination.txt")

	os.WriteFile(srcFile, []byte("content"), 0644)

	moveTool := &MoveTool{}
	moveReq := MoveRequest{
		Source:      srcFile,
		Destination: dstFile,
	}

	moveData, _ := json.Marshal(moveReq)
	_, err := moveTool.Execute(ctx, moveData)
	if err != nil {
		t.Fatalf("Move failed: %v", err)
	}

	if _, err := os.Stat(srcFile); err == nil {
		t.Error("Source file still exists")
	}

	if _, err := os.Stat(dstFile); err != nil {
		t.Errorf("Destination file not created: %v", err)
	}
}

func TestList(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	os.Create(filepath.Join(tempDir, "file1.txt"))
	os.Create(filepath.Join(tempDir, "file2.txt"))
	os.Mkdir(filepath.Join(tempDir, "subdir"), 0755)

	listTool := &ListTool{}
	listReq := ListRequest{
		Path: tempDir,
	}

	listData, _ := json.Marshal(listReq)
	result, err := listTool.Execute(ctx, listData)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	listResp := result.(ListResponse)
	if listResp.Count != 3 {
		t.Errorf("Expected 3 items, got %d", listResp.Count)
	}
}

func TestInfo(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	infoTool := &InfoTool{}
	infoReq := InfoRequest{
		Path: testFile,
	}

	infoData, _ := json.Marshal(infoReq)
	result, err := infoTool.Execute(ctx, infoData)
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}

	infoResp := result.(FileSystemInfo)
	if infoResp.Type != "file" {
		t.Errorf("Expected type 'file', got %q", infoResp.Type)
	}

	if infoResp.Size != 4 {
		t.Errorf("Expected size 4, got %d", infoResp.Size)
	}
}
