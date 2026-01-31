package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type MemoryStore struct {
	db *sql.DB
	mu sync.RWMutex
}

func NewMemoryStore(dbPath string) (*MemoryStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, err
	}

	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		return nil, err
	}

	store := &MemoryStore{db: db}
	if err := store.initSchema(); err != nil {
		return nil, err
	}

	if _, err := db.Exec(`DELETE FROM memories_fts WHERE name IN (SELECT name FROM memories WHERE deleted_at IS NOT NULL AND deleted_at < datetime('now', '-30 days'))`); err != nil {
	}
	result, err := db.Exec(`DELETE FROM memories WHERE deleted_at IS NOT NULL AND deleted_at < datetime('now', '-30 days')`)
	if err == nil {
		if rows, _ := result.RowsAffected(); rows > 0 {
			fmt.Printf("Purged %d soft-deleted memories older than 30 days\n", rows)
		}
	}

	return store, nil
}

func (s *MemoryStore) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS memories (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		content TEXT NOT NULL,
		category TEXT DEFAULT 'general',
		tags TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		accessed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		access_count INTEGER DEFAULT 0,
		deleted_at DATETIME
	);

	CREATE INDEX IF NOT EXISTS idx_memories_category ON memories(category);
	CREATE INDEX IF NOT EXISTS idx_memories_name ON memories(name);

	CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(name, content);
	`

	for _, stmt := range strings.Split(schema, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
}

func (s *MemoryStore) Create(id, name, content string, category Category, tags []string) (*Memory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM memories WHERE name = ? AND deleted_at IS NULL)", name).Scan(&exists)
	if err == nil && exists {
		return nil, fmt.Errorf("memory with name '%s' already exists", name)
	}

	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	memory := &Memory{
		ID:          id,
		Name:        name,
		Content:     content,
		Category:    category,
		Tags:        tags,
		CreatedAt:   now,
		UpdatedAt:   now,
		AccessedAt:  now,
		AccessCount: 0,
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(
		"INSERT INTO memories (id, name, content, category, tags, created_at, updated_at, accessed_at, access_count) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		id, name, content, category, string(tagsJSON), now, now, now, 0,
	)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	_, err = tx.Exec(
		"INSERT INTO memories_fts (name, content) VALUES (?, ?)",
		name, content,
	)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return memory, nil
}

func (s *MemoryStore) Read(identifier string) (*Memory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	row := s.db.QueryRow(
		"SELECT id, name, content, category, tags, created_at, updated_at, accessed_at, access_count, deleted_at FROM memories WHERE (id = ? OR name = ?) AND deleted_at IS NULL",
		identifier, identifier,
	)

	memory := &Memory{}
	var tagsJSON sql.NullString

	err := row.Scan(
		&memory.ID, &memory.Name, &memory.Content, &memory.Category, &tagsJSON,
		&memory.CreatedAt, &memory.UpdatedAt, &memory.AccessedAt, &memory.AccessCount, &memory.DeletedAt,
	)

	if err != nil {
		return nil, err
	}

	if tagsJSON.Valid {
		if err := json.Unmarshal([]byte(tagsJSON.String), &memory.Tags); err != nil {
			memory.Tags = []string{}
		}
	} else {
		memory.Tags = []string{}
	}

	_, err = s.db.Exec(
		"UPDATE memories SET accessed_at = ?, access_count = access_count + 1 WHERE id = ?",
		time.Now().UTC(), memory.ID,
	)

	return memory, nil
}

func (s *MemoryStore) Update(id, content string, tags []string) (*Memory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(
		"UPDATE memories SET content = ?, tags = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL",
		content, string(tagsJSON), now, id,
	)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	row := tx.QueryRow(
		"SELECT id, name, content, category, tags, created_at, updated_at, accessed_at, access_count FROM memories WHERE id = ?",
		id,
	)

	memory := &Memory{}
	var tagsJSONFromDB sql.NullString

	err = row.Scan(
		&memory.ID, &memory.Name, &memory.Content, &memory.Category, &tagsJSONFromDB,
		&memory.CreatedAt, &memory.UpdatedAt, &memory.AccessedAt, &memory.AccessCount,
	)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if tagsJSONFromDB.Valid {
		if err := json.Unmarshal([]byte(tagsJSONFromDB.String), &memory.Tags); err != nil {
			memory.Tags = []string{}
		}
	} else {
		memory.Tags = []string{}
	}

	_, err = tx.Exec(
		"DELETE FROM memories_fts WHERE name = ?",
		memory.Name,
	)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	_, err = tx.Exec(
		"INSERT INTO memories_fts (name, content) VALUES (?, ?)",
		memory.Name, content,
	)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return memory, nil
}

func (s *MemoryStore) UpdateFull(id, content string, category Category, tags []string) (*Memory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(
		"UPDATE memories SET content = ?, category = ?, tags = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL",
		content, category, string(tagsJSON), now, id,
	)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	row := tx.QueryRow(
		"SELECT id, name, content, category, tags, created_at, updated_at, accessed_at, access_count FROM memories WHERE id = ?",
		id,
	)

	memory := &Memory{}
	var tagsJSONFromDB sql.NullString

	err = row.Scan(
		&memory.ID, &memory.Name, &memory.Content, &memory.Category, &tagsJSONFromDB,
		&memory.CreatedAt, &memory.UpdatedAt, &memory.AccessedAt, &memory.AccessCount,
	)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if tagsJSONFromDB.Valid {
		if err := json.Unmarshal([]byte(tagsJSONFromDB.String), &memory.Tags); err != nil {
			memory.Tags = []string{}
		}
	} else {
		memory.Tags = []string{}
	}

	_, err = tx.Exec(
		"DELETE FROM memories_fts WHERE name = ?",
		memory.Name,
	)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	_, err = tx.Exec(
		"INSERT INTO memories_fts (name, content) VALUES (?, ?)",
		memory.Name, content,
	)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return memory, nil
}

func (s *MemoryStore) Delete(identifier string) (string, *time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	tx, err := s.db.Begin()
	if err != nil {
		return "", nil, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM memories_fts WHERE name = ? OR rowid IN (SELECT rowid FROM memories WHERE id = ?)`, identifier, identifier)
	if err != nil {
		return "", nil, err
	}

	result, err := tx.Exec(`DELETE FROM memories WHERE (id = ? OR name = ?)`, identifier, identifier)
	if err != nil {
		return "", nil, err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return "", nil, fmt.Errorf("memory '%s' not found", identifier)
	}

	if err := tx.Commit(); err != nil {
		return "", nil, err
	}

	return identifier, &now, nil
}

func (s *MemoryStore) List(category *Category, limit int) ([]*MemoryListItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := "SELECT id, name, category, content, created_at, accessed_at, access_count FROM memories WHERE deleted_at IS NULL"
	var args []interface{}

	if category != nil {
		query += " AND category = ?"
		args = append(args, *category)
	}

	query += " ORDER BY accessed_at DESC, created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*MemoryListItem

	for rows.Next() {
		item := &MemoryListItem{}
		var content string

		err := rows.Scan(
			&item.ID, &item.Name, &item.Category, &content,
			&item.CreatedAt, &item.AccessedAt, &item.AccessCount,
		)
		if err != nil {
			return nil, err
		}

		preview := truncate(content, 100)
		item.Preview = preview

		items = append(items, item)
	}

	return items, rows.Err()
}

func (s *MemoryStore) Search(query string, category *Category, limit int) ([]*SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sqlQuery := "SELECT m.id, m.name, m.category, m.content, m.created_at FROM memories m WHERE m.deleted_at IS NULL"
	var args []interface{}

	if query != "" {
		sqlQuery = fmt.Sprintf(
			"SELECT m.id, m.name, m.category, m.content, m.created_at FROM memories m "+
				"INNER JOIN memories_fts fts ON m.name = fts.name "+
				"WHERE fts.memories_fts MATCH ? AND m.deleted_at IS NULL",
		)
		args = append(args, query)

		if category != nil {
			sqlQuery += " AND m.category = ?"
			args = append(args, *category)
		}
	} else if category != nil {
		sqlQuery += " AND category = ?"
		args = append(args, *category)
	}

	sqlQuery += " LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*SearchResult

	for rows.Next() {
		result := &SearchResult{}
		var content string

		err := rows.Scan(
			&result.ID, &result.Name, &result.Category, &content, &result.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		result.Score = calculateRelevance(result.Name, content, query)
		result.Snippet = truncate(content, 150)

		results = append(results, result)
	}

	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results, rows.Err()
}

func (s *MemoryStore) Close() error {
	if _, err := s.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		// Checkpoint failure is not critical - DB will close normally even if truncation fails
	}
	return s.db.Close()
}

func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}

func calculateRelevance(name, content, query string) float64 {
	if query == "" {
		return 0
	}

	score := 0.0
	queryLower := strings.ToLower(query)
	nameLower := strings.ToLower(name)
	contentLower := strings.ToLower(content)

	if nameLower == queryLower {
		score += 10.0
	} else if strings.HasPrefix(nameLower, queryLower) {
		score += 8.0
	} else if strings.Contains(nameLower, queryLower) {
		score += 5.0
	}

	contentMatches := strings.Count(contentLower, queryLower)
	if contentMatches > 0 {
		score += float64(contentMatches)
	}

	return score
}
