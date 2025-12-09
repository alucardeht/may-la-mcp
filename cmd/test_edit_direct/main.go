package main

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"

    "github.com/maylamcp/mayla/internal/tools/files"
)

func main() {
    tmpDir, _ := os.MkdirTemp("", "edit-test-*")
    defer os.RemoveAll(tmpDir)
    
    testFile := filepath.Join(tmpDir, "test.txt")
    
    createTool := &files.CreateTool{}
    input, _ := json.Marshal(map[string]interface{}{
        "path": testFile,
        "type": "file",
    })
    createTool.Execute(input)
    
    writeTool := &files.WriteTool{}
    input, _ = json.Marshal(map[string]interface{}{
        "path":    testFile,
        "content": "Hello May-la MCP!\nLine 2\nLine 3",
    })
    writeTool.Execute(input)
    
    contentBefore, _ := os.ReadFile(testFile)
    fmt.Printf("BEFORE EDIT:\n%q\n\n", string(contentBefore))
    
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
    result, err := editTool.Execute(input)
    fmt.Printf("EDIT RESULT: %v, ERR: %v\n\n", result, err)
    
    contentAfter, _ := os.ReadFile(testFile)
    fmt.Printf("AFTER EDIT:\n%q\n\n", string(contentAfter))
    
    expected := "Hello MAYLA MCP!\nLine 2\nLine 3"
    if string(contentAfter) == expected {
        fmt.Println("SUCCESS - Content matches expected")
    } else {
        fmt.Printf("FAIL - Expected:\n%q\n", expected)
    }
}
