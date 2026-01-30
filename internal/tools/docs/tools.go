package docs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alucardeht/may-la-mcp/internal/tools"
)

func GetTools() []tools.Tool {
	return []tools.Tool{
		&DocWriteTool{},
		&DocReadTool{},
	}
}

type DocWriteTool struct{}

func (t *DocWriteTool) Name() string {
	return "doc_write"
}

func (t *DocWriteTool) Description() string {
	return `Write project documentation files (markdown, text, etc).

PURPOSE: Persistent documentation that lives WITH the project codebase.

WHEN TO USE doc_write:
- README files, CONTRIBUTING guides
- Architecture Decision Records (ADRs)
- API documentation
- Setup/installation guides
- Any docs that should be version-controlled with the project

WHEN TO USE memory_write INSTEAD:
- Cross-project learnings (applies to multiple projects)
- Personal conventions and preferences
- Knowledge that persists beyond any single project

PATH RESOLUTION:
- Relative paths resolve from project root (e.g., "docs/api.md")
- Absolute paths used as-is
- Auto-creates parent directories if needed`
}

func (t *DocWriteTool) Title() string {
	return "Write Project Documentation"
}

func (t *DocWriteTool) Annotations() map[string]bool {
	return tools.SafeWriteAnnotations()
}

func (t *DocWriteTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "File path relative to project root or absolute (required)"
			},
			"content": {
				"type": "string",
				"description": "File content (required)"
			},
			"project_root": {
				"type": "string",
				"description": "Project root for relative paths (optional - defaults to current directory)"
			}
		},
		"required": ["path", "content"]
	}`)
}

func (t *DocWriteTool) Execute(ctx context.Context, input json.RawMessage) (interface{}, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var req struct {
		Path        string `json:"path"`
		Content     string `json:"content"`
		ProjectRoot string `json:"project_root"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, err
	}

	if req.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	if req.Content == "" {
		return nil, fmt.Errorf("content is required")
	}

	projectRoot := req.ProjectRoot
	if projectRoot == "" {
		projectRoot = "."
	}

	var targetPath string
	if filepath.IsAbs(req.Path) {
		targetPath = req.Path
	} else {
		targetPath = filepath.Join(projectRoot, req.Path)
	}

	if !filepath.IsAbs(req.Path) {
		absRoot, err := filepath.Abs(projectRoot)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve project root: %w", err)
		}

		targetPath = filepath.Join(absRoot, req.Path)
		absTarget, err := filepath.Abs(targetPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve path: %w", err)
		}

		absRootCleaned := filepath.Clean(absRoot)
		absTargetCleaned := filepath.Clean(absTarget)

		if !isPathWithinRoot(absTargetCleaned, absRootCleaned) {
			return nil, fmt.Errorf("path escapes project root: %s", req.Path)
		}

		targetPath = absTargetCleaned
	}

	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}

	if err := os.WriteFile(targetPath, []byte(req.Content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"path":    targetPath,
		"size":    len(req.Content),
	}, nil
}

type DocReadTool struct{}

func (t *DocReadTool) Name() string {
	return "doc_read"
}

func (t *DocReadTool) Description() string {
	return `Read project documentation files.

Complements doc_write for reading documentation that lives with the project.
For cross-project knowledge, use memory_read instead.

PATH RESOLUTION:
- Relative paths resolve from project root
- Absolute paths used as-is`
}

func (t *DocReadTool) Title() string {
	return "Read Project Documentation"
}

func (t *DocReadTool) Annotations() map[string]bool {
	return tools.ReadOnlyAnnotations()
}

func (t *DocReadTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "File path relative to project root or absolute (required)"
			},
			"project_root": {
				"type": "string",
				"description": "Project root for relative paths (optional - defaults to current directory)"
			}
		},
		"required": ["path"]
	}`)
}

func (t *DocReadTool) Execute(ctx context.Context, input json.RawMessage) (interface{}, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	var req struct {
		Path        string `json:"path"`
		ProjectRoot string `json:"project_root"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, err
	}

	if req.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	projectRoot := req.ProjectRoot
	if projectRoot == "" {
		projectRoot = "."
	}

	var targetPath string
	if filepath.IsAbs(req.Path) {
		targetPath = req.Path
	} else {
		absRoot, err := filepath.Abs(projectRoot)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve project root: %w", err)
		}

		targetPath = filepath.Join(absRoot, req.Path)
		absTarget, err := filepath.Abs(targetPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve path: %w", err)
		}

		absRootCleaned := filepath.Clean(absRoot)
		absTargetCleaned := filepath.Clean(absTarget)

		if !isPathWithinRoot(absTargetCleaned, absRootCleaned) {
			return nil, fmt.Errorf("path escapes project root: %s", req.Path)
		}

		targetPath = absTargetCleaned
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return map[string]interface{}{
		"path":    targetPath,
		"content": string(content),
		"size":    len(content),
	}, nil
}

func isPathWithinRoot(targetPath, rootPath string) bool {
	rel, err := filepath.Rel(rootPath, targetPath)
	if err != nil {
		return false
	}

	return !filepath.HasPrefix(rel, "..")
}
