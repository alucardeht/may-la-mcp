package index

import "time"

type FileStatus string

const (
	StatusPending FileStatus = "pending"
	StatusIndexed FileStatus = "indexed"
	StatusFailed  FileStatus = "failed"
	StatusSkipped FileStatus = "skipped"
)

type IndexedFile struct {
	ID           int64      `json:"id"`
	Path         string     `json:"path"`
	ContentHash  string     `json:"content_hash"`
	Encoding     string     `json:"encoding"`
	Language     string     `json:"language"`
	Status       FileStatus `json:"status"`
	ErrorMessage string     `json:"error_message,omitempty"`
	IndexedAt    time.Time  `json:"indexed_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type IndexedSymbol struct {
	ID            int64  `json:"id"`
	FileID        int64  `json:"file_id"`
	Name          string `json:"name"`
	Kind          string `json:"kind"`
	Signature     string `json:"signature,omitempty"`
	LineStart     int    `json:"line_start"`
	LineEnd       int    `json:"line_end"`
	ColumnStart   int    `json:"column_start"`
	ColumnEnd     int    `json:"column_end"`
	Visibility    string `json:"visibility,omitempty"`
	Documentation string `json:"documentation,omitempty"`
	IsExported    bool   `json:"is_exported"`
}

type SymbolReference struct {
	ID       int64  `json:"id"`
	SymbolID int64  `json:"symbol_id"`
	FileID   int64  `json:"file_id"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Kind     string `json:"kind"`
	Context  string `json:"context,omitempty"`
}

type IndexStats struct {
	TotalFiles    int       `json:"total_files"`
	IndexedFiles  int       `json:"indexed_files"`
	FailedFiles   int       `json:"failed_files"`
	SkippedFiles  int       `json:"skipped_files"`
	TotalSymbols  int       `json:"total_symbols"`
	LastIndexedAt time.Time `json:"last_indexed_at"`
}

type IndexJob struct {
	Path     string
	Priority JobPriority
}

type JobPriority int

const (
	PriorityLow JobPriority = iota
	PriorityNormal
	PriorityHigh
)
