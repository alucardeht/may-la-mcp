package memory

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alucardeht/may-la-mcp/internal/tools"
)

func GetTools(dbPath string) ([]tools.Tool, error) {
	store, err := NewMemoryStore(dbPath)
	if err != nil {
		return nil, err
	}

	return []tools.Tool{
		NewMemoryWriteTool(store),
		NewMemoryReadTool(store),
		NewMemoryUpdateTool(store),
		NewMemoryListTool(store),
		NewMemorySearchTool(store),
		NewMemoryDeleteTool(store),
	}, nil
}

func GetToolsFromStore(store *MemoryStore) []tools.Tool {
	return []tools.Tool{
		NewMemoryWriteTool(store),
		NewMemoryReadTool(store),
		NewMemoryUpdateTool(store),
		NewMemoryListTool(store),
		NewMemorySearchTool(store),
		NewMemoryDeleteTool(store),
	}
}

type MemoryWriteTool struct {
	store *MemoryStore
}

func NewMemoryWriteTool(store *MemoryStore) *MemoryWriteTool {
	return &MemoryWriteTool{store: store}
}

func (t *MemoryWriteTool) Name() string {
	return "memory_write"
}

func (t *MemoryWriteTool) Description() string {
	return `Write content to persistent cross-project memory.

PURPOSE: Knowledge that persists across projects and sessions.

WHEN TO USE memory_write:
- Architectural patterns that apply to multiple projects
- Personal coding conventions and preferences
- Tool configurations and workflows
- Learnings from debugging sessions
- Patterns you want to remember across projects

WHEN TO USE doc_write INSTEAD:
- Project-specific documentation (README, API docs)
- Architecture Decision Records for THIS project
- Setup guides for THIS codebase
- Anything that should be version-controlled with the project

CATEGORIES:
- architecture: System design patterns
- conventions: Coding standards, naming patterns
- decisions: Why choices were made
- context: Background information
- general: General observations and notes`
}

func (t *MemoryWriteTool) Title() string {
	return "Write to Memory"
}

func (t *MemoryWriteTool) Annotations() map[string]bool {
	return tools.SafeWriteAnnotations()
}

func (t *MemoryWriteTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "Memory name/identifier"
			},
			"content": {
				"type": "string",
				"description": "Content to store"
			},
			"category": {
				"type": "string",
				"enum": ["architecture", "conventions", "decisions", "context", "general"],
				"description": "Memory category"
			},
			"tags": {
				"type": "array",
				"items": {"type": "string"},
				"description": "Tags for searchability"
			}
		},
		"required": ["name", "content"]
	}`)
}

func (t *MemoryWriteTool) Execute(ctx context.Context, input json.RawMessage) (interface{}, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var req struct {
		Name     string   `json:"name"`
		Content  string   `json:"content"`
		Category string   `json:"category"`
		Tags     []string `json:"tags"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, err
	}

	if req.Name == "" {
		return nil, fmt.Errorf("memory name is required")
	}

	if req.Content == "" {
		return nil, fmt.Errorf("memory content is required")
	}

	if req.Category == "" {
		req.Category = string(CategoryGeneral)
	}

	if req.Tags == nil {
		req.Tags = []string{}
	}

	id := generateID()
	memory, err := t.store.Create(id, req.Name, req.Content, Category(req.Category), req.Tags)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"id":      memory.ID,
		"name":    memory.Name,
		"path":    fmt.Sprintf("memory://%s/%s", req.Category, req.Name),
		"created": memory.CreatedAt,
	}, nil
}

type MemoryReadTool struct {
	store *MemoryStore
}

func NewMemoryReadTool(store *MemoryStore) *MemoryReadTool {
	return &MemoryReadTool{store: store}
}

func (t *MemoryReadTool) Name() string {
	return "memory_read"
}

func (t *MemoryReadTool) Description() string {
	return "Read content from persistent memory"
}

func (t *MemoryReadTool) Title() string {
	return "Read from Memory"
}

func (t *MemoryReadTool) Annotations() map[string]bool {
	return tools.ReadOnlyAnnotations()
}

func (t *MemoryReadTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "Memory name to read"
			}
		},
		"required": ["name"]
	}`)
}

func (t *MemoryReadTool) Execute(ctx context.Context, input json.RawMessage) (interface{}, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, err
	}

	if req.Name == "" {
		return nil, fmt.Errorf("memory name is required")
	}

	mem, err := t.store.Read(req.Name)
	if err != nil {
		return nil, fmt.Errorf("memory not found: %w", err)
	}

	if mem.Tags == nil {
		mem.Tags = []string{}
	}

	return map[string]interface{}{
		"id":            mem.ID,
		"name":          mem.Name,
		"content":       mem.Content,
		"category":      mem.Category,
		"tags":          mem.Tags,
		"created_at":    mem.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"updated_at":    mem.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"accessed_at":   mem.AccessedAt.Format("2006-01-02T15:04:05Z07:00"),
		"access_count":  mem.AccessCount,
	}, nil
}

type MemoryUpdateTool struct {
	store *MemoryStore
}

func NewMemoryUpdateTool(store *MemoryStore) *MemoryUpdateTool {
	return &MemoryUpdateTool{store: store}
}

func (t *MemoryUpdateTool) Name() string {
	return "memory_update"
}

func (t *MemoryUpdateTool) Description() string {
	return `Update existing memory content, category, or tags.

PARTIAL UPDATES: Only provided fields are updated. Omit fields to keep current values.

APPEND MODE: Set append=true to add content to end instead of replacing.

USE CASES:
- Add new learnings to existing memory
- Change category as understanding evolves
- Update tags for better searchability

For complete rewrites, use memory_write instead.`
}

func (t *MemoryUpdateTool) Title() string {
	return "Update Memory"
}

func (t *MemoryUpdateTool) Annotations() map[string]bool {
	return tools.SafeWriteAnnotations()
}

func (t *MemoryUpdateTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "Memory name/identifier (required)"
			},
			"content": {
				"type": "string",
				"description": "New content or content to append (optional - omit to keep current)"
			},
			"category": {
				"type": "string",
				"enum": ["architecture", "conventions", "decisions", "context", "general"],
				"description": "New category (optional - omit to keep current)"
			},
			"tags": {
				"type": "array",
				"items": {"type": "string"},
				"description": "New tags (optional - omit to keep current)"
			},
			"append": {
				"type": "boolean",
				"description": "If true, append content instead of replacing (default: false)"
			}
		},
		"required": ["name"]
	}`)
}

func (t *MemoryUpdateTool) Execute(ctx context.Context, input json.RawMessage) (interface{}, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	var req struct {
		Name     string   `json:"name"`
		Content  string   `json:"content"`
		Category string   `json:"category"`
		Tags     []string `json:"tags"`
		Append   bool     `json:"append"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, err
	}

	if req.Name == "" {
		return nil, fmt.Errorf("memory name is required")
	}

	existing, err := t.store.Read(req.Name)
	if err != nil {
		return nil, fmt.Errorf("memory not found: %w", err)
	}

	finalContent := existing.Content
	if req.Content != "" {
		if req.Append {
			finalContent = existing.Content + "\n" + req.Content
		} else {
			finalContent = req.Content
		}
	}

	finalTags := existing.Tags
	if len(req.Tags) > 0 {
		finalTags = req.Tags
	}

	finalCategory := existing.Category
	if req.Category != "" {
		finalCategory = Category(req.Category)
	}

	updated, err := t.store.UpdateFull(existing.ID, finalContent, finalCategory, finalTags)
	if err != nil {
		return nil, fmt.Errorf("failed to update memory: %w", err)
	}

	return map[string]interface{}{
		"success":    true,
		"id":         updated.ID,
		"name":       updated.Name,
		"content":    updated.Content,
		"category":   updated.Category,
		"tags":       updated.Tags,
		"updated_at": updated.UpdatedAt,
	}, nil
}

type MemoryListTool struct {
	store *MemoryStore
}

func NewMemoryListTool(store *MemoryStore) *MemoryListTool {
	return &MemoryListTool{store: store}
}

func (t *MemoryListTool) Name() string {
	return "memory_list"
}

func (t *MemoryListTool) Description() string {
	return "List all memories with optional filtering"
}

func (t *MemoryListTool) Title() string {
	return "List Memories"
}

func (t *MemoryListTool) Annotations() map[string]bool {
	return tools.ReadOnlyAnnotations()
}

func (t *MemoryListTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"category": {
				"type": "string",
				"description": "Filter by category"
			},
			"limit": {
				"type": "integer",
				"description": "Max results to return"
			}
		}
	}`)
}

func (t *MemoryListTool) Execute(ctx context.Context, input json.RawMessage) (interface{}, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	var req struct {
		Category string `json:"category"`
		Limit    int    `json:"limit"`
	}
	json.Unmarshal(input, &req)

	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 50
	}

	memories, err := t.store.List(categoryFromString(req.Category), req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list memories: %w", err)
	}

	items := make([]map[string]interface{}, 0, len(memories))
	for _, mem := range memories {
		items = append(items, map[string]interface{}{
			"id":            mem.ID,
			"name":          mem.Name,
			"category":      mem.Category,
			"preview":       mem.Preview,
			"created_at":    mem.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"accessed_at":   mem.AccessedAt.Format("2006-01-02T15:04:05Z07:00"),
			"access_count":  mem.AccessCount,
		})
	}

	return map[string]interface{}{
		"total":     len(memories),
		"memories":  items,
	}, nil
}

type MemorySearchTool struct {
	store *MemoryStore
}

func NewMemorySearchTool(store *MemoryStore) *MemorySearchTool {
	return &MemorySearchTool{store: store}
}

func (t *MemorySearchTool) Name() string {
	return "memory_search"
}

func (t *MemorySearchTool) Description() string {
	return "Search memories by content or tags"
}

func (t *MemorySearchTool) Title() string {
	return "Search Memories"
}

func (t *MemorySearchTool) Annotations() map[string]bool {
	return tools.ReadOnlyAnnotations()
}

func (t *MemorySearchTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "Search query"
			},
			"category": {
				"type": "string",
				"description": "Filter by category"
			},
			"limit": {
				"type": "integer",
				"description": "Max results"
			}
		},
		"required": ["query"]
	}`)
}

func (t *MemorySearchTool) Execute(ctx context.Context, input json.RawMessage) (interface{}, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	var req struct {
		Query    string `json:"query"`
		Category string `json:"category"`
		Limit    int    `json:"limit"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, err
	}

	if req.Query == "" {
		return nil, fmt.Errorf("search query is required")
	}

	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 50
	}

	results, err := t.store.Search(req.Query, categoryFromString(req.Category), req.Limit)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	items := make([]map[string]interface{}, 0, len(results))
	for _, result := range results {
		items = append(items, map[string]interface{}{
			"id":         result.ID,
			"name":       result.Name,
			"category":   result.Category,
			"score":      result.Score,
			"snippet":    result.Snippet,
			"created_at": result.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return map[string]interface{}{
		"query":   req.Query,
		"total":   len(results),
		"results": items,
	}, nil
}

type MemoryDeleteTool struct {
	store *MemoryStore
}

func NewMemoryDeleteTool(store *MemoryStore) *MemoryDeleteTool {
	return &MemoryDeleteTool{store: store}
}

func (t *MemoryDeleteTool) Name() string {
	return "memory_delete"
}

func (t *MemoryDeleteTool) Description() string {
	return "Delete a memory by name"
}

func (t *MemoryDeleteTool) Title() string {
	return "Delete Memory"
}

func (t *MemoryDeleteTool) Annotations() map[string]bool {
	return tools.DestructiveAnnotations()
}

func (t *MemoryDeleteTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "Memory name to delete"
			}
		},
		"required": ["name"]
	}`)
}

func (t *MemoryDeleteTool) Execute(ctx context.Context, input json.RawMessage) (interface{}, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, err
	}

	if req.Name == "" {
		return nil, fmt.Errorf("memory name is required")
	}

	identifier, deletedAt, err := t.store.Delete(req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to delete memory: %w", err)
	}

	if deletedAt == nil {
		return nil, fmt.Errorf("memory not found")
	}

	return map[string]interface{}{
		"success":    true,
		"identifier": identifier,
		"deleted_at": deletedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func generateID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x", b)
}

func categoryFromString(s string) *Category {
	if s == "" {
		return nil
	}
	cat := Category(s)
	return &cat
}
