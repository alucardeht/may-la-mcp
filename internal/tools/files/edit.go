package files

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/alucardeht/may-la-mcp/internal/tools"
)

type EditOperation struct {
	StartLine  int    `json:"startLine,omitempty"`
	EndLine    int    `json:"endLine,omitempty"`
	NewContent string `json:"newContent,omitempty"`
	Search     string `json:"search,omitempty"`
	Replace    string `json:"replace,omitempty"`
}

type EditRequest struct {
	Path  string          `json:"path"`
	Edits []EditOperation `json:"edits"`
}

type EditResponse struct {
	Path      string `json:"path"`
	Modified  bool   `json:"modified"`
	Size      int64  `json:"size"`
	Lines     int    `json:"lines"`
	EditsApplied int `json:"editsApplied"`
}

type EditTool struct{}

func (t *EditTool) Name() string {
	return "edit"
}

func (t *EditTool) Description() string {
	return "Edit file contents with multiple operations using line ranges or text search/replace"
}

func (t *EditTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Path to the file to edit (absolute path required)"
			},
			"edits": {
				"type": "array",
				"description": "Array of edit operations",
				"items": {
					"type": "object",
					"properties": {
						"startLine": {
							"type": "integer",
							"description": "Start line number (1-indexed)"
						},
						"endLine": {
							"type": "integer",
							"description": "End line number (1-indexed, inclusive)"
						},
						"newContent": {
							"type": "string",
							"description": "Replacement content"
						},
						"search": {
							"type": "string",
							"description": "Text to search for"
						},
						"replace": {
							"type": "string",
							"description": "Replacement text"
						}
					}
				},
				"minItems": 1
			}
		},
		"required": ["path", "edits"]
	}`)
}

func (t *EditTool) Execute(input json.RawMessage) (interface{}, error) {
	var req EditRequest
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if req.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	if len(req.Edits) == 0 {
		return nil, fmt.Errorf("at least one edit operation is required")
	}

	content, err := os.ReadFile(req.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	originalLines := make([]string, len(lines))
	copy(originalLines, lines)

	appliedCount := 0
	for _, edit := range req.Edits {
		if edit.Search != "" {
			for i := 0; i < len(lines); i++ {
				if strings.Contains(lines[i], edit.Search) {
					lines[i] = strings.ReplaceAll(lines[i], edit.Search, edit.Replace)
					appliedCount++
					break
				}
			}
		} else if edit.StartLine > 0 && edit.EndLine > 0 {
			if edit.StartLine < 1 || edit.EndLine < edit.StartLine || edit.EndLine > len(lines) {
				return nil, fmt.Errorf("invalid line range: %d-%d (file has %d lines)", edit.StartLine, edit.EndLine, len(lines))
			}

			startIdx := edit.StartLine - 1
			endIdx := edit.EndLine

			newLines := append([]string{}, lines[:startIdx]...)
			if edit.NewContent != "" {
				newLines = append(newLines, strings.Split(edit.NewContent, "\n")...)
			}
			newLines = append(newLines, lines[endIdx:]...)
			lines = newLines
			appliedCount++
		}
	}

	if len(lines) == 0 {
		lines = []string{""}
	}

	newContent := strings.Join(lines, "\n")
	if !strings.HasSuffix(newContent, "\n") && len(originalLines) > 0 && strings.Contains(string(content), "\n") {
		newContent += "\n"
	}

	tempPath := req.Path + ".tmp." + strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := os.WriteFile(tempPath, []byte(newContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := os.Rename(tempPath, req.Path); err != nil {
		os.Remove(tempPath)
		return nil, fmt.Errorf("failed to rename temp file: %w", err)
	}

	stat, err := os.Stat(req.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat edited file: %w", err)
	}
	finalLines := strings.Count(newContent, "\n")
	if newContent == "" {
		finalLines = 0
	}

	return EditResponse{
		Path:      req.Path,
		Modified:  newContent != string(content),
		Size:      stat.Size(),
		Lines:     finalLines,
		EditsApplied: appliedCount,
	}, nil
}

func (t *EditTool) Title() string {
	return "Edit File"
}

func (t *EditTool) Annotations() map[string]bool {
	return tools.SafeWriteAnnotations()
}
