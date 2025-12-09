package memory

import "time"

type Category string

const (
	CategoryArchitecture Category = "architecture"
	CategoryConventions  Category = "conventions"
	CategoryDecisions    Category = "decisions"
	CategoryContext      Category = "context"
	CategoryGeneral      Category = "general"
)

type Memory struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Content     string    `json:"content"`
	Category    Category  `json:"category"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	AccessedAt  time.Time `json:"accessed_at"`
	AccessCount int       `json:"access_count"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type SearchResult struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Category Category  `json:"category"`
	Score    float64   `json:"score"`
	Snippet  string    `json:"snippet"`
	CreatedAt time.Time `json:"created_at"`
}

type MemoryListItem struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Category Category  `json:"category"`
	Preview  string    `json:"preview"`
	CreatedAt time.Time `json:"created_at"`
	AccessedAt time.Time `json:"accessed_at"`
	AccessCount int     `json:"access_count"`
}
