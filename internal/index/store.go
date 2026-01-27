package index

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type IndexStore struct {
	db *sql.DB
	mu sync.RWMutex
}

func NewIndexStore(dbPath string) (*IndexStore, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create index dir: %w", err)
	}

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

	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, err
	}

	store := &IndexStore{db: db}
	if err := store.initSchema(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *IndexStore) initSchema() error {
	schema := GetSchema()

	lines := strings.Split(schema, "\n")
	var cleanLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "--") && trimmed != "" {
			cleanLines = append(cleanLines, line)
		}
	}
	cleanSchema := strings.Join(cleanLines, "\n")

	if _, err := s.db.Exec(cleanSchema); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	_, _ = s.db.Exec(`INSERT OR IGNORE INTO schema_version (version) VALUES (?)`, GetSchemaVersion())
	return nil
}

func (s *IndexStore) Close() error {
	return s.db.Close()
}

func (s *IndexStore) UpsertFile(file *IndexedFile) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	result, err := s.db.Exec(`
		INSERT INTO files (path, content_hash, encoding, language, status, error_message, indexed_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(path) DO UPDATE SET
			content_hash = excluded.content_hash,
			encoding = excluded.encoding,
			language = excluded.language,
			status = excluded.status,
			error_message = excluded.error_message,
			indexed_at = excluded.indexed_at,
			updated_at = CURRENT_TIMESTAMP
	`, file.Path, file.ContentHash, file.Encoding, file.Language, file.Status, file.ErrorMessage, now)

	if err != nil {
		return 0, fmt.Errorf("upsert file: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		row := s.db.QueryRow("SELECT id FROM files WHERE path = ?", file.Path)
		if err := row.Scan(&id); err != nil {
			return 0, fmt.Errorf("get file id: %w", err)
		}
	}

	return id, nil
}

func (s *IndexStore) GetFile(path string) (*IndexedFile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	file := &IndexedFile{}
	var indexedAt, updatedAt sql.NullTime
	var errorMsg sql.NullString

	err := s.db.QueryRow(`
		SELECT id, path, content_hash, encoding, language, status, error_message, indexed_at, updated_at
		FROM files WHERE path = ?
	`, path).Scan(
		&file.ID, &file.Path, &file.ContentHash, &file.Encoding, &file.Language,
		&file.Status, &errorMsg, &indexedAt, &updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get file: %w", err)
	}

	if errorMsg.Valid {
		file.ErrorMessage = errorMsg.String
	}
	if indexedAt.Valid {
		file.IndexedAt = indexedAt.Time
	}
	if updatedAt.Valid {
		file.UpdatedAt = updatedAt.Time
	}

	return file, nil
}

func (s *IndexStore) GetFileByID(id int64) (*IndexedFile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	file := &IndexedFile{}
	var indexedAt, updatedAt sql.NullTime
	var errorMsg sql.NullString

	err := s.db.QueryRow(`
		SELECT id, path, content_hash, encoding, language, status, error_message, indexed_at, updated_at
		FROM files WHERE id = ?
	`, id).Scan(
		&file.ID, &file.Path, &file.ContentHash, &file.Encoding, &file.Language,
		&file.Status, &errorMsg, &indexedAt, &updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get file by id: %w", err)
	}

	if errorMsg.Valid {
		file.ErrorMessage = errorMsg.String
	}
	if indexedAt.Valid {
		file.IndexedAt = indexedAt.Time
	}
	if updatedAt.Valid {
		file.UpdatedAt = updatedAt.Time
	}

	return file, nil
}

func (s *IndexStore) GetFilesByStatus(status FileStatus, limit int) ([]*IndexedFile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, path, content_hash, encoding, language, status, error_message, indexed_at, updated_at
		FROM files WHERE status = ? ORDER BY updated_at ASC LIMIT ?
	`, status, limit)

	if err != nil {
		return nil, fmt.Errorf("get files by status: %w", err)
	}
	defer rows.Close()

	var files []*IndexedFile

	for rows.Next() {
		file := &IndexedFile{}
		var indexedAt, updatedAt sql.NullTime
		var errorMsg sql.NullString

		err := rows.Scan(
			&file.ID, &file.Path, &file.ContentHash, &file.Encoding, &file.Language,
			&file.Status, &errorMsg, &indexedAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan file: %w", err)
		}

		if errorMsg.Valid {
			file.ErrorMessage = errorMsg.String
		}
		if indexedAt.Valid {
			file.IndexedAt = indexedAt.Time
		}
		if updatedAt.Valid {
			file.UpdatedAt = updatedAt.Time
		}

		files = append(files, file)
	}

	return files, rows.Err()
}

func (s *IndexStore) DeleteFile(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.Exec("DELETE FROM files WHERE path = ?", path)
	if err != nil {
		return fmt.Errorf("delete file: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil
	}

	return nil
}

func (s *IndexStore) UpdateFileStatus(path string, status FileStatus, errorMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	_, err := s.db.Exec(`
		UPDATE files SET status = ?, error_message = ?, updated_at = ? WHERE path = ?
	`, status, errorMsg, now, path)

	if err != nil {
		return fmt.Errorf("update file status: %w", err)
	}

	return nil
}

func (s *IndexStore) InsertSymbols(fileID int64, symbols []*IndexedSymbol) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM symbols WHERE file_id = ?", fileID)
	if err != nil {
		return fmt.Errorf("clear symbols: %w", err)
	}

	stmt, err := tx.Prepare(`
		INSERT INTO symbols (file_id, name, kind, signature, line_start, line_end, column_start, column_end, visibility, documentation, is_exported)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare stmt: %w", err)
	}
	defer stmt.Close()

	for _, sym := range symbols {
		_, err := stmt.Exec(
			fileID, sym.Name, sym.Kind, sym.Signature,
			sym.LineStart, sym.LineEnd, sym.ColumnStart, sym.ColumnEnd,
			sym.Visibility, sym.Documentation, sym.IsExported,
		)
		if err != nil {
			return fmt.Errorf("insert symbol %s: %w", sym.Name, err)
		}
	}

	return tx.Commit()
}

func (s *IndexStore) GetSymbolsByFile(fileID int64) ([]*IndexedSymbol, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, file_id, name, kind, signature, line_start, line_end, column_start, column_end, visibility, documentation, is_exported
		FROM symbols WHERE file_id = ? ORDER BY line_start ASC
	`, fileID)

	if err != nil {
		return nil, fmt.Errorf("get symbols by file: %w", err)
	}
	defer rows.Close()

	var symbols []*IndexedSymbol

	for rows.Next() {
		sym := &IndexedSymbol{}
		var signature, visibility, documentation sql.NullString
		var lineEnd, columnStart, columnEnd sql.NullInt64
		var isExported sql.NullInt64

		err := rows.Scan(
			&sym.ID, &sym.FileID, &sym.Name, &sym.Kind, &signature,
			&sym.LineStart, &lineEnd, &columnStart, &columnEnd,
			&visibility, &documentation, &isExported,
		)
		if err != nil {
			return nil, fmt.Errorf("scan symbol: %w", err)
		}

		if signature.Valid {
			sym.Signature = signature.String
		}
		if visibility.Valid {
			sym.Visibility = visibility.String
		}
		if documentation.Valid {
			sym.Documentation = documentation.String
		}
		if lineEnd.Valid {
			sym.LineEnd = int(lineEnd.Int64)
		}
		if columnStart.Valid {
			sym.ColumnStart = int(columnStart.Int64)
		}
		if columnEnd.Valid {
			sym.ColumnEnd = int(columnEnd.Int64)
		}
		if isExported.Valid {
			sym.IsExported = isExported.Int64 != 0
		}

		symbols = append(symbols, sym)
	}

	return symbols, rows.Err()
}

func (s *IndexStore) SearchSymbols(query string, limit int) ([]*IndexedSymbol, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT s.id, s.file_id, s.name, s.kind, s.signature, s.line_start, s.line_end,
		       s.column_start, s.column_end, s.visibility, s.documentation, s.is_exported
		FROM symbols s
		INNER JOIN symbols_fts fts ON s.id = fts.rowid
		WHERE symbols_fts MATCH ? LIMIT ?
	`, query, limit)

	if err != nil {
		return nil, fmt.Errorf("search symbols: %w", err)
	}
	defer rows.Close()

	var symbols []*IndexedSymbol

	for rows.Next() {
		sym := &IndexedSymbol{}
		var signature, visibility, documentation sql.NullString
		var lineEnd, columnStart, columnEnd sql.NullInt64
		var isExported sql.NullInt64

		err := rows.Scan(
			&sym.ID, &sym.FileID, &sym.Name, &sym.Kind, &signature,
			&sym.LineStart, &lineEnd, &columnStart, &columnEnd,
			&visibility, &documentation, &isExported,
		)
		if err != nil {
			return nil, fmt.Errorf("scan symbol: %w", err)
		}

		if signature.Valid {
			sym.Signature = signature.String
		}
		if visibility.Valid {
			sym.Visibility = visibility.String
		}
		if documentation.Valid {
			sym.Documentation = documentation.String
		}
		if lineEnd.Valid {
			sym.LineEnd = int(lineEnd.Int64)
		}
		if columnStart.Valid {
			sym.ColumnStart = int(columnStart.Int64)
		}
		if columnEnd.Valid {
			sym.ColumnEnd = int(columnEnd.Int64)
		}
		if isExported.Valid {
			sym.IsExported = isExported.Int64 != 0
		}

		symbols = append(symbols, sym)
	}

	return symbols, rows.Err()
}

func (s *IndexStore) GetSymbolByID(id int64) (*IndexedSymbol, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sym := &IndexedSymbol{}
	var signature, visibility, documentation sql.NullString
	var lineEnd, columnStart, columnEnd sql.NullInt64
	var isExported sql.NullInt64

	err := s.db.QueryRow(`
		SELECT id, file_id, name, kind, signature, line_start, line_end, column_start, column_end, visibility, documentation, is_exported
		FROM symbols WHERE id = ?
	`, id).Scan(
		&sym.ID, &sym.FileID, &sym.Name, &sym.Kind, &signature,
		&sym.LineStart, &lineEnd, &columnStart, &columnEnd,
		&visibility, &documentation, &isExported,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get symbol by id: %w", err)
	}

	if signature.Valid {
		sym.Signature = signature.String
	}
	if visibility.Valid {
		sym.Visibility = visibility.String
	}
	if documentation.Valid {
		sym.Documentation = documentation.String
	}
	if lineEnd.Valid {
		sym.LineEnd = int(lineEnd.Int64)
	}
	if columnStart.Valid {
		sym.ColumnStart = int(columnStart.Int64)
	}
	if columnEnd.Valid {
		sym.ColumnEnd = int(columnEnd.Int64)
	}
	if isExported.Valid {
		sym.IsExported = isExported.Int64 != 0
	}

	return sym, nil
}

func (s *IndexStore) ClearFileSymbols(fileID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("DELETE FROM symbols WHERE file_id = ?", fileID)
	if err != nil {
		return fmt.Errorf("clear file symbols: %w", err)
	}

	return nil
}

func (s *IndexStore) InsertReferences(symbolID int64, refs []*SymbolReference) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM symbol_refs WHERE symbol_id = ?", symbolID)
	if err != nil {
		return fmt.Errorf("clear references: %w", err)
	}

	stmt, err := tx.Prepare(`
		INSERT INTO symbol_refs (symbol_id, file_id, line, column, kind, context)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare stmt: %w", err)
	}
	defer stmt.Close()

	for _, ref := range refs {
		_, err := stmt.Exec(
			symbolID, ref.FileID, ref.Line, ref.Column, ref.Kind, ref.Context,
		)
		if err != nil {
			return fmt.Errorf("insert reference: %w", err)
		}
	}

	return tx.Commit()
}

func (s *IndexStore) GetReferencesForSymbol(symbolID int64) ([]*SymbolReference, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, symbol_id, file_id, line, column, kind, context
		FROM symbol_refs WHERE symbol_id = ? ORDER BY file_id ASC, line ASC
	`, symbolID)

	if err != nil {
		return nil, fmt.Errorf("get references for symbol: %w", err)
	}
	defer rows.Close()

	var refs []*SymbolReference

	for rows.Next() {
		ref := &SymbolReference{}
		var column sql.NullInt64
		var ctxStr sql.NullString

		err := rows.Scan(
			&ref.ID, &ref.SymbolID, &ref.FileID, &ref.Line, &column, &ref.Kind, &ctxStr,
		)
		if err != nil {
			return nil, fmt.Errorf("scan reference: %w", err)
		}

		if column.Valid {
			ref.Column = int(column.Int64)
		}
		if ctxStr.Valid {
			ref.Context = ctxStr.String
		}

		refs = append(refs, ref)
	}

	return refs, rows.Err()
}

func (s *IndexStore) GetReferencesInFile(fileID int64) ([]*SymbolReference, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, symbol_id, file_id, line, column, kind, context
		FROM symbol_refs WHERE file_id = ? ORDER BY line ASC
	`, fileID)

	if err != nil {
		return nil, fmt.Errorf("get references in file: %w", err)
	}
	defer rows.Close()

	var refs []*SymbolReference

	for rows.Next() {
		ref := &SymbolReference{}
		var column sql.NullInt64
		var ctxStr sql.NullString

		err := rows.Scan(
			&ref.ID, &ref.SymbolID, &ref.FileID, &ref.Line, &column, &ref.Kind, &ctxStr,
		)
		if err != nil {
			return nil, fmt.Errorf("scan reference: %w", err)
		}

		if column.Valid {
			ref.Column = int(column.Int64)
		}
		if ctxStr.Valid {
			ref.Context = ctxStr.String
		}

		refs = append(refs, ref)
	}

	return refs, rows.Err()
}

func (s *IndexStore) GetStats() (*IndexStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &IndexStats{}

	err := s.db.QueryRow(`
		SELECT
			COUNT(*) as total_files,
			COALESCE(SUM(CASE WHEN status = 'indexed' THEN 1 ELSE 0 END), 0) as indexed_files,
			COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) as failed_files,
			COALESCE(SUM(CASE WHEN status = 'skipped' THEN 1 ELSE 0 END), 0) as skipped_files,
			MAX(indexed_at) as last_indexed_at
		FROM files
	`).Scan(&stats.TotalFiles, &stats.IndexedFiles, &stats.FailedFiles, &stats.SkippedFiles, &stats.LastIndexedAt)

	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}

	err = s.db.QueryRow("SELECT COUNT(*) FROM symbols").Scan(&stats.TotalSymbols)
	if err != nil {
		return nil, fmt.Errorf("get symbol count: %w", err)
	}

	return stats, nil
}
