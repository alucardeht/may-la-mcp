package memory

import (
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
		NewMemoryListTool(store),
		NewMemorySearchTool(store),
		NewMemoryDeleteTool(store),
	}, nil
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
	return "Write content to persistent memory"
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
				"enum": ["architecture", "conventions", "decisions", "context", "notes"],
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

func (t *MemoryWriteTool) Execute(input json.RawMessage) (interface{}, error) {
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

func (t *MemoryReadTool) Execute(input json.RawMessage) (interface{}, error) {
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

func (t *MemoryListTool) Execute(input json.RawMessage) (interface{}, error) {
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

func (t *MemorySearchTool) Execute(input json.RawMessage) (interface{}, error) {
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

func (t *MemoryDeleteTool) Execute(input json.RawMessage) (interface{}, error) {
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
